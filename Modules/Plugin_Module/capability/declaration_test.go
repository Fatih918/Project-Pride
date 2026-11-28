package capability

import (
	"errors"
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
	cases := map[string]string{
		"reserved declaration namespace": `"id":"io.pride/text.generate"`,
		"invalid capability identifier": `"id":"Com.example/document.summarize"`,
		"invalid contract version": `"contract_version":"v1.2"`,
		"unsupported invocation mode": `"invocation_mode":"io.pride.invocation/stream"`,
		"input schema traversal": `"input_schema_ref":"schemas/../secret.json"`,
		"absolute output schema": `"output_schema_ref":"/schemas/output.json"`,
		"invalid pride version": `"pride_version":"^1.0.0"`,
		"invalid runtime profile": `"runtime_profile_ids":["com.example/worker"]`,
		"self dependency": `"capability_id":"com.example/document.summarize"`,
		"invalid dependency version": `"version_requirement":"~1.2"`,
		"invalid dependency kind": `"kind":"recommended"`,
	}
	for name, replacement := range cases {
		t.Run(name, func(t *testing.T) {
			source := validDeclaration
			switch name {
			case "reserved declaration namespace", "invalid capability identifier":
				source = replaceOnce(source, `"id":"com.example/document.summarize"`, replacement)
			case "invalid contract version":
				source = replaceOnce(source, `"contract_version":"1.2.3"`, replacement)
			case "unsupported invocation mode":
				source = replaceOnce(source, `"invocation_mode":"io.pride.invocation/request-response"`, replacement)
			case "input schema traversal":
				source = replaceOnce(source, `"input_schema_ref":"schemas/input.schema.json"`, replacement)
			case "absolute output schema":
				source = replaceOnce(source, `"output_schema_ref":"schemas/output.schema.json"`, replacement)
			case "invalid pride version":
				source = replaceOnce(source, `"pride_version":">=1.0.0 <2.0.0"`, replacement)
			case "invalid runtime profile":
				source = replaceOnce(source, `"runtime_profile_ids":["io.pride.runtime/worker.standard"]`, replacement)
			default:
				source = replaceOnce(source, `"capability_id":"org.acme/document.parse"`, stringsReplaceCapabilityID(replacement, name))
				if name == "invalid dependency version" {
					source = replaceOnce(source, `"version_requirement":">=2.1.0 <3.0.0"`, replacement)
				}
				if name == "invalid dependency kind" {
					source = replaceOnce(source, `"kind":"required"`, replacement)
				}
			}
			if _, err := ValidateDeclaration([]byte(source)); !errors.Is(err, ErrInvalidDeclaration) &&
				!errors.Is(err, ErrInvalidCompatibility) && !errors.Is(err, ErrInvalidDependency) {
				t.Fatalf("ValidateDeclaration() error = %v, want semantic validation failure", err)
			}
		})
	}
}

func TestValidateDeclarationRejectsUnknownFieldsAndDuplicateDependencies(t *testing.T) {
	unknown := replaceOnce(validDeclaration, `"name":"Document Summary",`, `"name":"Document Summary","unknown":true,`)
	if _, err := ValidateDeclaration([]byte(unknown)); !errors.Is(err, ErrInvalidDeclaration) {
		t.Fatalf("unknown field error = %v", err)
	}

	duplicate := replaceOnce(validDeclaration, `}]
}`, `},
		{"capability_id":"org.acme/document.parse","version_requirement":">=2.1.0 <3.0.0","kind":"optional"}]
}`)
	if _, err := ValidateDeclaration([]byte(duplicate)); !errors.Is(err, ErrInvalidDependency) {
		t.Fatalf("duplicate dependency error = %v", err)
	}
}

func replaceOnce(source, old, replacement string) string {
	return strings.Replace(source, old, replacement, 1)
}

func stringsReplaceCapabilityID(replacement, name string) string {
	if name == "self dependency" {
		return replacement
	}
	return `"capability_id":"invalid"`
}
