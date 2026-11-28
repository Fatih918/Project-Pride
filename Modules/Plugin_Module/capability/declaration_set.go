package capability

import "errors"

const MaxCapabilityDeclarations = 256

var ErrInvalidDeclarationSet = errors.New("invalid capability declaration set")

// ValidateDeclarationSet validates set-level invariants after individual
// declarations have passed ValidateDeclaration. It does not resolve
// dependencies, compose Plugins, or authorize capability invocation.
func ValidateDeclarationSet(declarations []CapabilityDeclaration) error {
	if len(declarations) > MaxCapabilityDeclarations {
		return ErrInvalidDeclarationSet
	}

	seen := make(map[string]struct{}, len(declarations))
	for _, declaration := range declarations {
		if err := ValidateID(declaration.ID); err != nil {
			return ErrInvalidDeclarationSet
		}
		if _, exists := seen[declaration.ID]; exists {
			return ErrInvalidDeclarationSet
		}
		seen[declaration.ID] = struct{}{}
	}
	return nil
}
