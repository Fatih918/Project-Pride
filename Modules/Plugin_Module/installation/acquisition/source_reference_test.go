package acquisition

import (
	"errors"
	"testing"

	"pride-plugin-module/domain"
)

func TestParseRegistryReleaseReference(t *testing.T) {
	input := "pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1"
	got, err := ParseRegistryReleaseReference(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Namespace != "community" || got.PluginID != domain.PluginID("example.plugin") ||
		got.ReleaseID != "rel_01J9K8M7N6P5Q4R3S2T1" {
		t.Fatalf("unexpected parsed reference: %#v", got)
	}
	if got.String() != input {
		t.Fatalf("canonical representation = %q, want %q", got.String(), input)
	}
}

func TestParseRegistryReleaseReferenceRejectsInvalidInputs(t *testing.T) {
	cases := []string{
		"",
		"https://registry.example/plugins/example.plugin",
		"pride-registry:v2/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1",
		"pride-registry:v1/community/example.plugin",
		"pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1/extra",
		"pride-registry:v1//example.plugin/rel_01J9K8M7N6P5Q4R3S2T1",
		"pride-registry:v1/community/../rel_01J9K8M7N6P5Q4R3S2T1",
		"pride-registry:v1/community/example.plugin/latest",
		"pride-registry:v1/community/example.plugin/rel%2Fother",
		"pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1?token=secret",
		" pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1",
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseRegistryReleaseReference(input); !errors.Is(err, ErrInvalidSourceReference) {
				t.Fatalf("expected invalid source reference, got %v", err)
			}
		})
	}
}

func TestNewDescriptorResolutionRequestBindsExpectedIdentity(t *testing.T) {
	ref, err := ParseRegistryReleaseReference("pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1")
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewDescriptorResolutionRequest(ref, domain.PluginID("example.plugin"), "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if request.Reference != ref || request.ExpectedPluginID != ref.PluginID || request.ExpectedVersion != "1.2.3" {
		t.Fatalf("unexpected request: %#v", request)
	}
}

func TestNewDescriptorResolutionRequestRejectsIdentityMismatch(t *testing.T) {
	ref, err := ParseRegistryReleaseReference("pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewDescriptorResolutionRequest(ref, domain.PluginID("other.plugin"), "1.2.3"); !errors.Is(err, ErrInvalidResolutionRequest) {
		t.Fatalf("expected identity mismatch rejection, got %v", err)
	}
}
