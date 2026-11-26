package startup

import (
	"context"
	"errors"
	"testing"

	"pride-plugin-module/registry"
)

type lifecycleRecoveryFunc func(context.Context) error

func (f lifecycleRecoveryFunc) Recover(ctx context.Context) error { return f(ctx) }

func TestInitializeWithLifecycleRunsBothRecoveriesBeforeDiscovery(t *testing.T) {
	var order []string
	installation := recoveryFunc(func(context.Context) error {
		order = append(order, "installation")
		return nil
	})
	lifecycle := lifecycleRecoveryFunc(func(context.Context) error {
		order = append(order, "lifecycle")
		return nil
	})
	service, err := InitializeWithLifecycle(context.Background(), installation, lifecycle, registry.NewMemoryRepository())
	if err != nil { t.Fatalf("initialize: %v", err) }
	if service == nil { t.Fatal("expected discovery service") }
	if len(order) != 2 || order[0] != "installation" || order[1] != "lifecycle" {
		t.Fatalf("unexpected recovery order: %#v", order)
	}
}

func TestInitializeWithLifecycleFailsClosedWhenLifecycleRecoveryFails(t *testing.T) {
	recoveryErr := errors.New("lifecycle journal unavailable")
	service, err := InitializeWithLifecycle(
		context.Background(),
		recoveryFunc(func(context.Context) error { return nil }),
		lifecycleRecoveryFunc(func(context.Context) error { return recoveryErr }),
		registry.NewMemoryRepository(),
	)
	if !errors.Is(err, recoveryErr) { t.Fatalf("expected lifecycle recovery error, got %v", err) }
	if service != nil { t.Fatal("discovery must not be exposed after lifecycle recovery failure") }
}

func TestInitializeWithLifecycleRejectsMissingLifecycleRecoverer(t *testing.T) {
	service, err := InitializeWithLifecycle(context.Background(), recoveryFunc(func(context.Context) error { return nil }), nil, registry.NewMemoryRepository())
	if !errors.Is(err, ErrMissingDependency) { t.Fatalf("expected missing dependency error, got %v", err) }
	if service != nil { t.Fatal("expected nil service") }
}
