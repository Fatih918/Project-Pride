package conformance_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"pride-plugin-module/capability"
	capabilityschema "pride-plugin-module/capability/schema"
)

const fixtureVersion = "open-capability/v1"

func TestOpenCapabilityV1PositiveFixtures(t *testing.T) {
	moduleRoot := moduleRoot(t)
	fixtureRoot := filepath.Join(moduleRoot, "conformance", fixtureVersion)
	schemaRoot := filepath.Join(moduleRoot, "schemas")

	descriptorValidator := compileSchemaFile(t, filepath.Join(schemaRoot, "registry-artifact-descriptor-v2.schema.json"))
	metadataValidator := compileSchemaFile(t, filepath.Join(schemaRoot, "platform-metadata-v2.schema.json"))
	declarationValidator := compileSchemaFile(t, filepath.Join(schemaRoot, "capability-declaration.schema.json"))

	descriptor := readFixture(t, filepath.Join(fixtureRoot, "positive", "descriptor-v2.json"))
	if err := descriptorValidator.Validate(descriptor); err != nil {
		t.Fatalf("positive descriptor rejected: %v", err)
	}

	var decoded struct {
		PlatformMetadata struct {
			Capabilities []json.RawMessage `json:"capabilities"`
		} `json:"platform_metadata"`
	}
	if err := json.Unmarshal(descriptor, &decoded); err != nil {
		t.Fatalf("decode positive descriptor: %v", err)
	}

	metadata, err := json.Marshal(extractPlatformMetadata(t, descriptor))
	if err != nil {
		t.Fatalf("marshal platform metadata: %v", err)
	}
	if err := metadataValidator.Validate(metadata); err != nil {
		t.Fatalf("positive platform metadata rejected: %v", err)
	}

	declarations := make([]capability.CapabilityDeclaration, 0, len(decoded.PlatformMetadata.Capabilities))
	for index, raw := range decoded.PlatformMetadata.Capabilities {
		if err := declarationValidator.Validate(raw); err != nil {
			t.Fatalf("positive capability %d failed structural validation: %v", index, err)
		}
		declaration, err := capability.ValidateDeclaration(raw)
		if err != nil {
			t.Fatalf("positive capability %d failed semantic validation: %v", index, err)
		}
		declarations = append(declarations, declaration)
	}
	if err := capability.ValidateDeclarationSet(declarations); err != nil {
		t.Fatalf("positive capability set rejected: %v", err)
	}

	requestSchema := compileSchemaFile(t, filepath.Join(fixtureRoot, "positive", "request.schema.json"))
	responseSchema := compileSchemaFile(t, filepath.Join(fixtureRoot, "positive", "response.schema.json"))
	if err := requestSchema.Validate([]byte(`{"text":"A document to summarize"}`)); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if err := responseSchema.Validate([]byte(`{"summary":"A concise summary"}`)); err != nil {
		t.Fatalf("valid response rejected: %v", err)
	}
	if err := requestSchema.Validate([]byte(`{"unexpected":"field"}`)); !errors.Is(err, capabilityschema.ErrInvalidInstance) {
		t.Fatalf("invalid request error = %v, want ErrInvalidInstance", err)
	}
}

func TestOpenCapabilityV1NegativeFixtures(t *testing.T) {
	moduleRoot := moduleRoot(t)
	fixtureRoot := filepath.Join(moduleRoot, "conformance", fixtureVersion)
	descriptorValidator := compileSchemaFile(t, filepath.Join(moduleRoot, "schemas", "registry-artifact-descriptor-v2.schema.json"))

	tests := []struct {
		name string
		run  func([]byte) error
	}{
		{
			name: "descriptor-wrong-schema-version.json",
			run: func(data []byte) error {
				return descriptorValidator.Validate(data)
			},
		},
		{
			name: "capability-id-uppercase.json",
			run: func(data []byte) error {
				_, err := capability.ValidateDeclaration(data)
				return err
			},
		},
		{
			name: "capability-schema-reference-traversal.json",
			run: func(data []byte) error {
				_, err := capability.ValidateDeclaration(data)
				return err
			},
		},
		{
			name: "capability-unsupported-invocation-mode.json",
			run: func(data []byte) error {
				_, err := capability.ValidateDeclaration(data)
				return err
			},
		},
		{
			name: "invalid-contract-version.json",
			run: func(data []byte) error {
				_, err := capability.ValidateDeclaration(data)
				return err
			},
		},
		{
			name: "capability-unknown-property.json",
			run: func(data []byte) error {
				_, err := capability.ValidateDeclaration(data)
				return err
			},
		},
		{
			name: "schema-unsupported-pattern-keyword.json",
			run: func(data []byte) error {
				_, err := capabilityschema.Compile(data)
				return err
			},
		},
		{
			name: "duplicate-json-keys.json",
			run: func(data []byte) error {
				_, err := capabilityschema.Compile(data)
				return err
			},
		},
		{
			name: "duplicate-capability-id.json",
			run: func(data []byte) error {
				var fixture struct {
					Capabilities []struct {
						ID string `json:"id"`
					} `json:"capabilities"`
				}
				if err := json.Unmarshal(data, &fixture); err != nil {
					return err
				}
				declarations := make([]capability.CapabilityDeclaration, 0, len(fixture.Capabilities))
				for _, item := range fixture.Capabilities {
					declarations = append(declarations, capability.CapabilityDeclaration{ID: item.ID})
				}
				return capability.ValidateDeclarationSet(declarations)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := readFixture(t, filepath.Join(fixtureRoot, "negative", test.name))
			if err := test.run(data); err == nil {
				t.Fatalf("negative fixture %q was accepted", test.name)
			}
		})
	}
}

func compileSchemaFile(t *testing.T, path string) *capabilityschema.Validator {
	t.Helper()
	document, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema %q: %v", path, err)
	}
	validator, err := capabilityschema.Compile(document)
	if err != nil {
		t.Fatalf("compile schema %q: %v", path, err)
	}
	return validator
}

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	document, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", path, err)
	}
	if len(document) == 0 || len(document) > capabilityschema.MaxDocumentBytes {
		t.Fatalf("fixture %q has invalid size %d", path, len(document))
	}
	return document
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve conformance test source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", "..", ".."))
}

func extractPlatformMetadata(t *testing.T, descriptor []byte) any {
	t.Helper()
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(descriptor, &envelope); err != nil {
		t.Fatalf("decode descriptor envelope: %v", err)
	}
	raw, ok := envelope["platform_metadata"]
	if !ok {
		t.Fatal("descriptor has no platform_metadata")
	}
	var metadata any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		t.Fatalf("decode platform metadata: %v", err)
	}
	return metadata
}
