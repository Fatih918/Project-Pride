package implementation

import (
	"context"
	"fmt"
	executioncontract "pride-ai-module/contracts/execution"
	"testing"
	"sync"
)

type testImplementation struct {
	name string
}

func (i *testImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
	}, nil
}

func TestResolverResolve(t *testing.T) {
	expected := &testImplementation{
		name: "test",
	}

	resolver := NewResolver()

	if err := resolver.Register(
		"test.implementation",
		expected,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	actual, err := resolver.Resolve(
		"test.implementation",
	)
	if err != nil {
		t.Fatalf(
			"Resolve() failed: %v",
			err,
		)
	}

	if actual != expected {
		t.Fatal(
			"Resolve() returned unexpected implementation",
		)
	}
}

func TestResolverRejectsUnknownReference(t *testing.T) {
	expected := &testImplementation{
		name: "test",
	}
	resolver := NewResolver()

	if err := resolver.Register(
		"test.implementation",
		expected,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	_, err := resolver.Resolve(
		"unknown.implementation",
	)

	if err == nil {
		t.Fatal(
			"expected unknown implementation reference to return an error",
		)
	}
}

func TestResolverRejectsEmptyReference(t *testing.T) {
	expected := &testImplementation{
		name: "test",
	}
	resolver := NewResolver()

	if err := resolver.Register(
		"test.implementation",
		expected,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}
	_, err := resolver.Resolve("")

	if err == nil {
		t.Fatal(
			"expected empty implementation reference to return an error",
		)
	}
}

func TestResolverRejectsNilImplementation(t *testing.T) {
	resolver := NewResolver()

	expected := &testImplementation{
		name: "test",
	}
	if err := resolver.Register(
		"test.implementation",
		expected,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	_, err := resolver.Resolve(
		"nil.implementation",
	)

	if err == nil {
		t.Fatal(
			"expected nil implementation to return an error",
		)
	}
}
func TestResolverRegisterAndResolve(t *testing.T) {
	resolver := NewResolver()

	first := &testImplementation{
		name: "first",
	}

	second := &testImplementation{
		name: "second",
	}

	if err := resolver.Register(
		"provider.first",
		first,
	); err != nil {
		t.Fatalf(
			"Register(first) failed: %v",
			err,
		)
	}

	if err := resolver.Register(
		"provider.second",
		second,
	); err != nil {
		t.Fatalf(
			"Register(second) failed: %v",
			err,
		)
	}

	resolvedFirst, err := resolver.Resolve(
		"provider.first",
	)
	if err != nil {
		t.Fatalf(
			"Resolve(first) failed: %v",
			err,
		)
	}

	resolvedSecond, err := resolver.Resolve(
		"provider.second",
	)
	if err != nil {
		t.Fatalf(
			"Resolve(second) failed: %v",
			err,
		)
	}

	if resolvedFirst != first {
		t.Fatal(
			"first implementation was not resolved correctly",
		)
	}

	if resolvedSecond != second {
		t.Fatal(
			"second implementation was not resolved correctly",
		)
	}
}

func TestResolverRegisterRejectsDuplicateReference(
	t *testing.T,
) {
	resolver := NewResolver()

	implementation := &testImplementation{
		name: "test",
	}

	if err := resolver.Register(
		"test.implementation",
		implementation,
	); err != nil {
		t.Fatalf(
			"first Register() failed: %v",
			err,
		)
	}

	if err := resolver.Register(
		"test.implementation",
		implementation,
	); err == nil {
		t.Fatal(
			"expected duplicate registration to fail",
		)
	}
}

func TestResolverRegisterRejectsEmptyReference(
	t *testing.T,
) {
	resolver := NewResolver()

	err := resolver.Register(
		"",
		&testImplementation{},
	)

	if err == nil {
		t.Fatal(
			"expected empty reference to be rejected",
		)
	}
}

func TestResolverRegisterRejectsNilImplementation(
	t *testing.T,
) {
	resolver := NewResolver()

	err := resolver.Register(
		"nil.implementation",
		nil,
	)

	if err == nil {
		t.Fatal(
			"expected nil implementation to be rejected",
		)
	}
}

func TestResolverRegisterAfterConstruction(
	t *testing.T,
) {
	resolver := NewResolver()

	for i := 0; i < 10; i++ {
		reference := fmt.Sprintf(
			"custom.implementation.%d",
			i,
		)

		err := resolver.Register(
			reference,
			&testImplementation{
				name: reference,
			},
		)

		if err != nil {
			t.Fatalf(
				"Register(%q) failed: %v",
				reference,
				err,
			)
		}
	}

	for i := 0; i < 10; i++ {
		reference := fmt.Sprintf(
			"custom.implementation.%d",
			i,
		)

		if _, err := resolver.Resolve(reference); err != nil {
			t.Fatalf(
				"Resolve(%q) failed: %v",
				reference,
				err,
			)
		}
	}
}
func TestResolverConcurrentRegisterAndResolve(t *testing.T) {
	t.Parallel()

	resolver := NewResolver()
	const (
		workers       = 20
		registrations = 100
		resolveLoops  = 500
	)

	seed := &testImplementation{name: "seed"}
	if err := resolver.Register("concurrent.implementation.seed", seed); err != nil {
		t.Fatalf("register seed implementation: %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup

	// Writers register unique implementations while readers resolve an
	// already-registered implementation. Running both groups together
	// exercises concurrent map reads and writes under the race detector.
	wg.Add(workers * 2)
	for worker := 0; worker < workers; worker++ {
		worker := worker

		go func() {
			defer wg.Done()
			<-start

			for i := 0; i < registrations; i++ {
				reference := fmt.Sprintf("concurrent.implementation.%d.%d", worker, i)
				implementation := &testImplementation{name: reference}

				if err := resolver.Register(reference, implementation); err != nil {
					t.Errorf("Register(%q) failed: %v", reference, err)
					continue
				}

				resolved, err := resolver.Resolve(reference)
				if err != nil {
					t.Errorf("Resolve(%q) after registration failed: %v", reference, err)
					continue
				}
				if resolved != implementation {
					t.Errorf("Resolve(%q) returned a different implementation", reference)
				}
			}
		}()
	}

	for worker := 0; worker < workers; worker++ {
		go func() {
			defer wg.Done()
			<-start

			for i := 0; i < resolveLoops; i++ {
				resolved, err := resolver.Resolve("concurrent.implementation.seed")
				if err != nil {
					t.Errorf("Resolve(seed) failed: %v", err)
					return
				}
				if resolved != seed {
					t.Error("Resolve(seed) returned a different implementation")
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}
