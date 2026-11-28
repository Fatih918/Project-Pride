package capability

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"unicode/utf8"

	"pride-plugin-module/capability/schema"
)

const (
	MaxCapabilityNameBytes         = 128
	MaxCapabilityDescriptionBytes  = 1024
	MaxCapabilityDependencies      = 64
	MaxRuntimeProfileIDs           = 32
)

var (
	ErrInvalidDeclaration     = errors.New("invalid capability declaration")
	ErrInvalidCompatibility = errors.New("invalid capability platform compatibility")
	ErrInvalidDependency    = errors.New("invalid capability dependency")
)

// CapabilityDeclaration is the validated, typed form of one signed capability
// declaration. It is metadata only; validation does not authorize invocation.
type CapabilityDeclaration struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	Description           string                 `json:"description"`
	ContractVersion       string                 `json:"contract_version"`
	InvocationMode        string                 `json:"invocation_mode"`
	InputSchemaRef        string                 `json:"input_schema_ref"`
	OutputSchemaRef       string                 `json:"output_schema_ref"`
	PlatformCompatibility *PlatformCompatibility `json:"platform_compatibility,omitempty"`
	Dependencies          []CapabilityDependency `json:"dependencies,omitempty"`
}

// PlatformCompatibility contains declarative PRIDE/runtime constraints.
type PlatformCompatibility struct {
	PrideVersion      string   `json:"pride_version,omitempty"`
	RuntimeProfileIDs []string `json:"runtime_profile_ids,omitempty"`
}

// CapabilityDependency is a declarative requirement, not an invocation edge.
type CapabilityDependency struct {
	CapabilityID       string `json:"capability_id"`
	VersionRequirement string `json:"version_requirement"`
	Kind               string `json:"kind"`
}

// ValidateDeclaration validates the POCS-0001 declaration schema and its
// cross-field semantic constraints. It performs no package or network I/O.
func ValidateDeclaration(document []byte) (CapabilityDeclaration, error) {
	var declaration CapabilityDeclaration

	compiled := declarationValidator()
	if declarationSchemaErr != nil {
		return declaration, fmt.Errorf("%w: compile embedded declaration schema: %v", ErrInvalidDeclaration, declarationSchemaErr)
	}
	if err := compiled.Validate(document); err != nil {
		return declaration, fmt.Errorf("%w: %v", ErrInvalidDeclaration, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(document))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&declaration); err != nil {
		return CapabilityDeclaration{}, fmt.Errorf("%w: %v", ErrInvalidDeclaration, err)
	}

	if err := ValidateID(declaration.ID); err != nil {
		return CapabilityDeclaration{}, fmt.Errorf("%w: %v", ErrInvalidDeclaration, err)
	}
	if !validHumanText(declaration.Name, MaxCapabilityNameBytes, false) ||
		!validHumanText(declaration.Description, MaxCapabilityDescriptionBytes, true) {
		return CapabilityDeclaration{}, ErrInvalidDeclaration
	}
	if _, err := ParseVersion(declaration.ContractVersion); err != nil {
		return CapabilityDeclaration{}, fmt.Errorf("%w: %v", ErrInvalidDeclaration, err)
	}
	if declaration.InvocationMode != "io.pride.invocation/request-response" {
		return CapabilityDeclaration{}, ErrInvalidDeclaration
	}
	if err := schema.ValidateReference(declaration.InputSchemaRef); err != nil {
		return CapabilityDeclaration{}, fmt.Errorf("%w: input schema reference: %v", ErrInvalidDeclaration, err)
	}
	if err := schema.ValidateReference(declaration.OutputSchemaRef); err != nil {
		return CapabilityDeclaration{}, fmt.Errorf("%w: output schema reference: %v", ErrInvalidDeclaration, err)
	}

	if declaration.PlatformCompatibility != nil {
		if err := validatePlatformCompatibility(*declaration.PlatformCompatibility); err != nil {
			return CapabilityDeclaration{}, err
		}
	}
	if err := validateDependencies(declaration.ID, declaration.Dependencies); err != nil {
		return CapabilityDeclaration{}, err
	}
	return declaration, nil
}

func validHumanText(value string, maxBytes int, allowEmpty bool) bool {
	if !utf8.ValidString(value) || len([]byte(value)) > maxBytes || (!allowEmpty && len(value) == 0) {
		return false
	}
	for _, r := range value {
		if (r < 0x20 || r == 0x7f) && r != 9 && r != 10 && r != 13 {
			return false
		}
	}
	return true
}

func validatePlatformCompatibility(value PlatformCompatibility) error {
	if value.PrideVersion == "" && len(value.RuntimeProfileIDs) == 0 {
		return ErrInvalidCompatibility
	}
	if value.PrideVersion != "" {
		if _, err := ParseVersionRequirement(value.PrideVersion); err != nil {
			return fmt.Errorf("%w: pride_version: %v", ErrInvalidCompatibility, err)
		}
	}
	if len(value.RuntimeProfileIDs) > MaxRuntimeProfileIDs {
		return ErrInvalidCompatibility
	}
	seen := make(map[string]struct{}, len(value.RuntimeProfileIDs))
	for _, id := range value.RuntimeProfileIDs {
		if err := ValidateRuntimeProfileID(id); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidCompatibility, err)
		}
		if _, exists := seen[id]; exists {
			return ErrInvalidCompatibility
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateDependencies(ownerID string, dependencies []CapabilityDependency) error {
	if len(dependencies) > MaxCapabilityDependencies {
		return ErrInvalidDependency
	}
	seen := make(map[string]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		if err := validateCanonicalID(dependency.CapabilityID); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidDependency, err)
		}
		if dependency.CapabilityID == ownerID {
			return ErrInvalidDependency
		}
		if _, err := ParseVersionRequirement(dependency.VersionRequirement); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidDependency, err)
		}
		if dependency.Kind != "required" && dependency.Kind != "optional" {
			return ErrInvalidDependency
		}
		if _, exists := seen[dependency.CapabilityID]; exists {
			return ErrInvalidDependency
		}
		seen[dependency.CapabilityID] = struct{}{}
	}
	return nil
}

// capabilityDeclarationSchema is embedded so validation does not depend on a
// mutable working-directory path or package filesystem lookup.
var (
	declarationSchemaOnce      sync.Once
	declarationSchemaValidator *schema.Validator
	declarationSchemaErr       error
)

func declarationValidator() *schema.Validator {
	declarationSchemaOnce.Do(func() {
		declarationSchemaValidator, declarationSchemaErr = schema.Compile(capabilityDeclarationSchema)
	})
	return declarationSchemaValidator
}

var capabilityDeclarationSchema = []byte(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["id", "name", "description", "contract_version", "invocation_mode", "input_schema_ref", "output_schema_ref"],
  "properties": {
    "id": {"type": "string", "minLength": 1, "maxLength": 255},
    "name": {"type": "string", "minLength": 1, "maxLength": 128},
    "description": {"type": "string", "maxLength": 1024},
    "contract_version": {"type": "string", "minLength": 5, "maxLength": 64},
    "invocation_mode": {"const": "io.pride.invocation/request-response"},
    "input_schema_ref": {"type": "string", "minLength": 9, "maxLength": 240},
    "output_schema_ref": {"type": "string", "minLength": 9, "maxLength": 240},
    "platform_compatibility": {"$ref": "#/$defs/platformCompatibility"},
    "dependencies": {"type": "array", "maxItems": 64, "items": {"$ref": "#/$defs/dependency"}}
  },
  "$defs": {
    "platformCompatibility": {
      "type": "object",
      "properties": {
        "pride_version": {"type": "string", "minLength": 1, "maxLength": 128},
        "runtime_profile_ids": {"type": "array", "maxItems": 32, "uniqueItems": true, "items": {"type": "string", "minLength": 1, "maxLength": 128}}
      },
      "anyOf": [{"required": ["pride_version"]}, {"required": ["runtime_profile_ids"]}],
      "additionalProperties": false
    },
    "dependency": {
      "type": "object",
      "required": ["capability_id", "version_requirement", "kind"],
      "properties": {
        "capability_id": {"type": "string", "minLength": 1, "maxLength": 255},
        "version_requirement": {"type": "string", "minLength": 1, "maxLength": 128},
        "kind": {"enum": ["required", "optional"]}
      },
      "additionalProperties": false
    }
  },
  "additionalProperties": false
}`)
