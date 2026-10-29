package provider

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

type testProvider struct{ name string }

func (testProvider) Execute(
	ctx context.Context,
	request Request,
) (Response, error) {
	return Response{
		Output: request.Input,
	}, nil
}

func TestResolverRegisterAndResolve(t *testing.T) {
	resolver := NewResolver()
	testProvider := testProvider{}

	err := resolver.Register("test.provider", testProvider)
	if err != nil {
		t.Fatalf("expected register to succeed, got error: %v", err)
	}

	resolved, err := resolver.Resolve("test.provider")
	if err != nil {
		t.Fatalf("expected resolve to succeed, got error: %v", err)
	}

	if resolved == nil {
		t.Fatal("expected resolved provider, got nil")
	}

	if resolved != testProvider {
		t.Fatal("expected resolved provider to match registered provider")
	}
}

func TestResolverRegisterRejectsEmptyReference(t *testing.T) {
	resolver := NewResolver()

	err := resolver.Register("", testProvider{})
	if err == nil {
		t.Fatal("expected error for empty provider reference")
	}
}

func TestResolverRegisterRejectsNilProvider(t *testing.T) {
	resolver := NewResolver()

	err := resolver.Register("test.provider", nil)
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
}

func TestResolverRegisterRejectsDuplicateReference(t *testing.T) {
	resolver := NewResolver()
	testProvider := testProvider{}

	err := resolver.Register("test.provider", testProvider)
	if err != nil {
		t.Fatalf("expected first register to succeed, got error: %v", err)
	}

	err = resolver.Register("test.provider", testProvider)
	if err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
}

func TestResolverResolveRejectsEmptyReference(t *testing.T) {
	resolver := NewResolver()

	_, err := resolver.Resolve("")
	if err == nil {
		t.Fatal("expected error for empty provider reference")
	}
}

func TestResolverResolveUnknownReference(t *testing.T) {
	resolver := NewResolver()

	_, err := resolver.Resolve("unknown.provider")
	if err == nil {
		t.Fatal("expected error for unknown provider reference")
	}
}

func TestResolverNilReceiver(t *testing.T) {
	var resolver *Resolver

	if err := resolver.Register("test.provider", testProvider{}); err == nil {
		t.Fatal("expected error when registering with nil resolver")
	}

	if _, err := resolver.Resolve("test.provider"); err == nil {
		t.Fatal("expected error when resolving with nil resolver")
	}
}

func TestResolverResolvesCorrectProvider(t *testing.T) {
	resolver := NewResolver()

	first := testProvider{}
	second := testProvider{}

	if err := resolver.Register("provider.first", first); err != nil {
		t.Fatalf("register first provider: %v", err)
	}

	if err := resolver.Register("provider.second", second); err != nil {
		t.Fatalf("register second provider: %v", err)
	}

	resolvedFirst, err := resolver.Resolve("provider.first")
	if err != nil {
		t.Fatalf("resolve first provider: %v", err)
	}

	resolvedSecond, err := resolver.Resolve("provider.second")
	if err != nil {
		t.Fatalf("resolve second provider: %v", err)
	}

	if resolvedFirst != first {
		t.Fatal("expected first reference to resolve to first provider")
	}

	if resolvedSecond != second {
		t.Fatal("expected second reference to resolve to second provider")
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

	seed := testProvider{name: "seed"}
	if err := resolver.Register("concurrent.provider.seed", seed); err != nil {
		t.Fatalf("register seed provider: %v", err)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(workers * 2)

	// Writers register unique providers while readers resolve the seeded
	// provider concurrently. This is intended to run under go test -race.
	for worker := 0; worker < workers; worker++ {
		worker := worker

		go func() {
			defer wg.Done()
			<-start

			for i := 0; i < registrations; i++ {
				ref := fmt.Sprintf("concurrent.provider.%d.%d", worker, i)
				p := testProvider{name: ref}

				if err := resolver.Register(ref, p); err != nil {
					t.Errorf("Register(%q) failed: %v", ref, err)
					continue
				}

				resolved, err := resolver.Resolve(ref)
				if err != nil {
					t.Errorf("Resolve(%q) after registration failed: %v", ref, err)
					continue
				}
				if resolved != p {
					t.Errorf("Resolve(%q) returned a different provider", ref)
				}
			}
		}()
	}

	for worker := 0; worker < workers; worker++ {
		go func() {
			defer wg.Done()
			<-start

			for i := 0; i < resolveLoops; i++ {
				resolved, err := resolver.Resolve("concurrent.provider.seed")
				if err != nil {
					t.Errorf("Resolve(seed) failed: %v", err)
					return
				}
				if resolved != seed {
					t.Error("Resolve(seed) returned a different provider")
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
}
