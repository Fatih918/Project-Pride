package schema

import (
	"errors"
	"strings"
	"testing"
)

func compileTestSchema(t *testing.T, source string) *Validator {
	t.Helper()
	v, err := Compile([]byte(source))
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return v
}

func TestCompileAndValidateSupportedKeywords(t *testing.T) {
	v := compileTestSchema(t, `{
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"type":"object",
		"properties":{
			"name":{"type":"string","minLength":2,"maxLength":8},
			"age":{"type":"integer","minimum":13,"maximum":99},
			"score":{"type":"number","multipleOf":0.25},
			"tags":{"type":"array","items":{"type":"string"},"minItems":1,"maxItems":3,"uniqueItems":true}
		},
		"required":["name","age"],
		"additionalProperties":false
	}`)
	if err := v.Validate([]byte(`{"name":"Razin","age":15,"score":1.25,"tags":["go","systems"]}`)); err != nil {
		t.Fatalf("valid instance rejected: %v", err)
	}
	invalid := []string{
		`{"name":"R","age":15}`,
		`{"name":"Razin","age":12}`,
		`{"name":"Razin","age":15,"extra":true}`,
		`{"name":"Razin","age":15,"tags":["go","go"]}`,
		`{"name":"Razin","age":15,"score":1.1}`,
	}
	for _, instance := range invalid {
		if err := v.Validate([]byte(instance)); !errors.Is(err, ErrInvalidInstance) {
			t.Errorf("Validate(%s) = %v, want ErrInvalidInstance", instance, err)
		}
	}
}

func TestCompileRejectsUnsupportedAndMalformedSchemas(t *testing.T) {
	cases := []string{
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","pattern":"x"}`,
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"x":true}}`,
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":true}`,
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","$ref":"https://example.com/schema.json"}`,
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","$ref":"#/$defs/missing"}`,
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","type":"string"}`,
	}
	for _, source := range cases {
		if _, err := Compile([]byte(source)); err == nil {
			t.Errorf("Compile(%s) succeeded, want failure", source)
		}
	}
}

func TestCompileAndValidateLocalDefinitionReference(t *testing.T) {
	v := compileTestSchema(t, `{
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"type":"object",
		"properties":{"payload":{"$ref":"#/$defs/payload"}},
		"$defs":{"payload":{"type":"object","required":["ok"],"properties":{"ok":{"const":true}},"additionalProperties":false}}
	}`)
	if err := v.Validate([]byte(`{"payload":{"ok":true}}`)); err != nil {
		t.Fatal(err)
	}
	if err := v.Validate([]byte(`{"payload":{"ok":false}}`)); !errors.Is(err, ErrInvalidInstance) {
		t.Fatalf("error = %v, want ErrInvalidInstance", err)
	}
}

func TestCompileRejectsDuplicateKeysAndUnsupportedRoot(t *testing.T) {
	for _, source := range []string{
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","type":"string"}`,
		`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","properties":{"x":{"type":"string","type":"number"}}}`,
		`true`,
		`{"$schema":"https://json-schema.org/draft/2020-12/schema"}`,
	} {
		if _, err := Compile([]byte(source)); err == nil {
			t.Errorf("Compile(%s) succeeded", source)
		}
	}
}

func TestValidateRejectsDuplicateInstanceKeysAndBounds(t *testing.T) {
	v := compileTestSchema(t, `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}`)
	if err := v.Validate([]byte(`{"x":1,"x":2}`)); !errors.Is(err, ErrInvalidInstance) {
		t.Fatalf("duplicate key error = %v", err)
	}
	if err := v.Validate([]byte(`{"x":"` + strings.Repeat("a", MaxStringBytes+1) + `"}`)); !errors.Is(err, ErrInvalidInstance) {
		t.Fatalf("oversized string error = %v", err)
	}
}

func TestApplicators(t *testing.T) {
	v := compileTestSchema(t, `{
		"$schema":"https://json-schema.org/draft/2020-12/schema",
		"type":"integer",
		"allOf":[{"minimum":1},{"maximum":10}],
		"not":{"const":5}
	}`)
	for _, instance := range []string{"1", "3", "10"} {
		if err := v.Validate([]byte(instance)); err != nil {
			t.Errorf("%s rejected: %v", instance, err)
		}
	}
	for _, instance := range []string{"0", "5", "11"} {
		if err := v.Validate([]byte(instance)); !errors.Is(err, ErrInvalidInstance) {
			t.Errorf("%s error=%v", instance, err)
		}
	}
}
