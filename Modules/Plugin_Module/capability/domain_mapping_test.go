package capability

import (
	"testing"

	"pride-plugin-module/domain"
)

func TestToDomainCapabilityContractsPreservesVersionedMetadata(t *testing.T) {
	declarations := []CapabilityDeclaration{{
		ID: "acme/text.generate", Name: "Text", Description: "Generate text",
		ContractVersion: "1.2.3", InvocationMode: "io.pride.invocation/request-response",
		InputSchemaRef: "schemas/text-input.json", OutputSchemaRef: "schemas/text-output.json",
		PlatformCompatibility: &PlatformCompatibility{
			PrideVersion: ">=1.0.0", RuntimeProfileIDs: []string{"io.pride.runtime/worker"},
		},
		Dependencies: []CapabilityDependency{{
			CapabilityID: "acme/model.infer", VersionRequirement: ">=2.0.0 <3.0.0", Kind: "required",
		}},
	}}

	got := ToDomainCapabilityContracts(declarations)
	if len(got) != 1 {
		t.Fatalf("expected one contract, got %d", len(got))
	}
	contract := got[0]
	if contract.ContractVersion != "1.2.3" || contract.InputSchemaRef != "schemas/text-input.json" ||
		contract.OutputSchemaRef != "schemas/text-output.json" || contract.InvocationMode != declarations[0].InvocationMode {
		t.Fatalf("contract metadata was not preserved: %#v", contract)
	}
	if contract.PlatformCompatibility == nil || contract.PlatformCompatibility.PrideVersion != ">=1.0.0" ||
		len(contract.PlatformCompatibility.RuntimeProfileIDs) != 1 {
		t.Fatalf("platform compatibility was not preserved: %#v", contract.PlatformCompatibility)
	}
	if len(contract.Dependencies) != 1 || contract.Dependencies[0].VersionRequirement != ">=2.0.0 <3.0.0" {
		t.Fatalf("dependency metadata was not preserved: %#v", contract.Dependencies)
	}
}

func TestPluginCloneDeepCopiesCapabilityContracts(t *testing.T) {
	plugin := domain.Plugin{Protected: domain.ProtectedMetadata{CapabilityContracts: []domain.CapabilityContract{{
		ID: "acme/text.generate",
		PlatformCompatibility: &domain.CapabilityPlatformCompatibility{
			RuntimeProfileIDs: []string{"io.pride.runtime/worker"},
		},
		Dependencies: []domain.CapabilityDependency{{CapabilityID: "acme/model.infer"}},
	}}}}
	clone := plugin.Clone()
	clone.Protected.CapabilityContracts[0].PlatformCompatibility.RuntimeProfileIDs[0] = "changed"
	clone.Protected.CapabilityContracts[0].Dependencies[0].CapabilityID = "changed"
	if plugin.Protected.CapabilityContracts[0].PlatformCompatibility.RuntimeProfileIDs[0] != "io.pride.runtime/worker" ||
		plugin.Protected.CapabilityContracts[0].Dependencies[0].CapabilityID != "acme/model.infer" {
		t.Fatal("Plugin.Clone shared nested capability contract memory")
	}
}
