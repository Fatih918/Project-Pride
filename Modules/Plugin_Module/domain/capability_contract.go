package domain

// cloneCapabilityContracts deep-copies nested contract metadata.
func cloneCapabilityContracts(input []CapabilityContract) []CapabilityContract {
	if input == nil {
		return nil
	}
	output := make([]CapabilityContract, len(input))
	for i, contract := range input {
		output[i] = contract
		if contract.PlatformCompatibility != nil {
			compatibility := *contract.PlatformCompatibility
			compatibility.RuntimeProfileIDs = append([]string(nil), contract.PlatformCompatibility.RuntimeProfileIDs...)
			output[i].PlatformCompatibility = &compatibility
		}
		output[i].Dependencies = append([]CapabilityDependency(nil), contract.Dependencies...)
	}
	return output
}
