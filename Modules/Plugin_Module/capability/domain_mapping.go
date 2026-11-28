package capability

import "pride-plugin-module/domain"

// ToDomainCapabilityContracts maps already validated declarations into the
// SDK-neutral Plugin domain model. Callers must only pass declarations obtained
// from ValidateDeclaration after the containing Registry descriptor and
// artifact have passed cryptographic verification.
func ToDomainCapabilityContracts(declarations []CapabilityDeclaration) []domain.CapabilityContract {
	if declarations == nil {
		return nil
	}
	contracts := make([]domain.CapabilityContract, len(declarations))
	for i, declaration := range declarations {
		contract := domain.CapabilityContract{
			ID:              declaration.ID,
			Name:            declaration.Name,
			Description:     declaration.Description,
			ContractVersion: declaration.ContractVersion,
			InvocationMode:  declaration.InvocationMode,
			InputSchemaRef:  declaration.InputSchemaRef,
			OutputSchemaRef: declaration.OutputSchemaRef,
			Dependencies:    make([]domain.CapabilityDependency, len(declaration.Dependencies)),
		}
		if declaration.PlatformCompatibility != nil {
			contract.PlatformCompatibility = &domain.CapabilityPlatformCompatibility{
				PrideVersion:      declaration.PlatformCompatibility.PrideVersion,
				RuntimeProfileIDs: append([]string(nil), declaration.PlatformCompatibility.RuntimeProfileIDs...),
			}
		}
		for j, dependency := range declaration.Dependencies {
			contract.Dependencies[j] = domain.CapabilityDependency{
				CapabilityID:       dependency.CapabilityID,
				VersionRequirement: dependency.VersionRequirement,
				Kind:               dependency.Kind,
			}
		}
		contracts[i] = contract
	}
	return contracts
}
