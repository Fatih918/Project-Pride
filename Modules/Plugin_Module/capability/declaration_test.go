package capability

import (
	"errors"
	"strings"
	"testing"
)

const validDeclaration = `{
	"id":"com.example/document.summarize",
	"name":"Document Summary",
	"description":"Summarizes documents",
	"contract_version":"1.2.3",
	"invocation_mode":"io.pride.invocation/request-response",
	"input_schema_ref":"schemas/input.schema.json",
	"output_schema_ref":"schemas/output.schema.json",
	"platform_compatibility":{
		"pride_version":">=1.0.0 <2.0.0",
		"runtime_profile_ids":["io.pride.runtime/worker.standard"]
	},
	"dependencies":[{
		"capability_id":"org.acme/document.parse",
		"version_requirement":">=2.1.0 <3.0.0",
		"kind":"required"
	}]
}`

func TestValidateDeclarationAcceptsValidDeclaration(t *testing.T) {
	declaration, err := ValidateDeclaration([]byte(validDeclaration))
	if err != nil {
		t.Fatalf("ValidateDeclaration() error = %v", err)
	}
	if declaration.ID != "com.example/document.summarize" {
		t.Fatalf("ID = %q", declaration.ID)
	}
	if len(declaration.Dependencies) != 1 || declaration.Dependencies[0].Kind != "required" {
		t.Fatalf("dependencies = %#v", declaration.Dependencies)
	}
}

func TestValidateDeclarationRejectsInvalidSemanticFields(t *testing.T) {
	cases := []struct {
		name string
		old  string
		new  string
	}{
		{"reserved declaration namespace", `"id":"com.example/document.summarize"`, `"id":"io.pride/text.generate"`},
		{"invalid capability identifier", `"id":"com.example/document.summarize"`, `"id":"Com.example/document.summarize"`},
		{"invalid contract version", `"contract_version":"1.2.3"`, `"contract_version":"v1.2"`},
		{"unsupported invocation mode", `"invocation_mode":"io.pride.invocation/request-response"`, `"invocation_mode":"io.pride.invocation/stream"`},
		{"input schema traversal", `"input_schema_ref":"schemas/input.schema.json"`, `"input_schema_ref":"schemas/../secret.json"`},
		{"absolute output schema", `"output_schema_ref":"schemas/output.schema.json"`, `"output_schema_ref":"/schemas/output.json"`},
		{"invalid pride version", `"pride_version":">=1.0.0 <2.0.0"`, `"pride_version":"^1.0.0"`},
		{"invalid runtime profile", `"runtime_profile_ids":["io.pride.runtime/worker.standard"]`, `"runtime_profile_ids":["com.example/worker"]`},
		{"self dependency", `"capability_id":"org.acme/document.parse"`, `"capability_id":"com.example/document.summarize"`},
		{"invalid dependency version", `"version_requirement":">=2.1.0 <3.0.0"`, `"version_requirement":"~1.2"`},
		{"invalid dependency kind", `"kind":"required"`, `"kind":"recommended"`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(validDeclaration, test.old, test.new, 1)
			if source == validDeclaration {
				t.Fatal("test mutation did not apply")
			}
			if _, err := ValidateDeclaration([]byte(source)); !errors.Is(err, ErrInvalidDeclaration) &&
				!errors.Is(err, ErrInvalidCompatibility) && !errors.Is(err, ErrInvalidDependency) {
				t.Fatalf("ValidateDeclaration() error = %v, want semantic validation failure", err)
			}
		})
	}
}

func TestValidateDeclarationRejectsUnknownFieldsAndDuplicateDependencies(t *testing.T) {
	unknown := strings.Replace(validDeclaration, `"name":"Document Summary",`, `"name":"Document Summary","unknown":true,`, 1)
	if _, err := ValidateDeclaration([]byte(unknown)); !errors.Is(err, ErrInvalidDeclaration) {
		t.Fatalf("unknown field error = %v", err)
	}

	duplicate := strings.Replace(validDeclaration, `}]
}`, `},
		{"capability_id":"org.acme/document.parse","version_requirement":">=2.1.0 <3.0.0","kind":"optional"}]
}`, 1)
	if _, err := ValidateDeclaration([]byte(duplicate)); !errors.Is(err, ErrInvalidDependency) {
		t.Fatalf("duplicate dependency error = %v", err)
	}
}

func TestValidateDeclarationAllowsReferenceToPlatformOwnedCapability(t *testing.T) {
	source := strings.Replace(validDeclaration, "org.acme/document.parse", "io.pride/text.generate", 1)
	if _, err := ValidateDeclaration([]byte(source)); err != nil {
		t.Fatalf("platform-owned dependency reference rejected: %v", err)
	}
}
