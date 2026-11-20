package startup

import (
	"context"
	"errors"
	"testing"

	"pride-plugin-module/registry"
)

type recoveryFunc func(context.Context) error

func (f recoveryFunc) Recover(ctx context.Context) error { return f(ctx) }

func TestInitializeRunsRecoveryBeforeReturningDiscovery(t *testing.T) {
	repository := registry.NewMemoryRepository()
	recovered := false

	service, err := Initialize(context.Background(), recoveryFunc(func(context.Context) error {
		recovered = true
		return nil
	}), repository)
	if err != nil {
		t.Fatalf("initialize startup: %v", err)
	}
	if service == nil {
		t.Fatal("expected discovery service")
	}
	if !recovered {
		t.Fatal("discovery service returned before recovery completed")
	}
}

func TestInitializeFailsClosedWhenRecoveryFails(t *testing.T) {
	recoveryErr := errors.New("trust service unavailable")
	service, err := Initialize(context.Background(), recoveryFunc(func(context.Context) error {
		return recoveryErr
	}), registry.NewMemoryRepository())

	if !errors.Is(err, recoveryErr) {
		t.Fatalf("expected recovery error, got %v", err)
	}
	if service != nil {
		t.Fatal("discovery service must not be returned after recovery failure")
	}
}

func TestInitializeRejectsMissingDependenciesBeforeRecovery(t *testing.T) {
	called := false
	recoverer := recoveryFunc(func(context.Context) error {
		called = true
		return nil
	})

	service, err := Initialize(context.Background(), recoverer, nil)
	if !errors.Is(err, ErrMissingDependency) {
		t.Fatalf("expected missing dependency error, got %v", err)
	}
	if service != nil {
		t.Fatal("expected nil discovery service")
	}
	if called {
		t.Fatal("recovery must not run with missing dependencies")
	}
}

func TestInitializeHonorsCanceledContextBeforeRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false

	service, err := Initialize(ctx, recoveryFunc(func(context.Context) error {
		called = true
		return nil
	}), registry.NewMemoryRepository())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if service != nil {
		t.Fatal("expected nil discovery service")
	}
	if called {
		t.Fatal("recovery must not run with canceled context")
	}
}
