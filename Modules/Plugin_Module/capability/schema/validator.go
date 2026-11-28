package schema

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	MaxSchemaDepth          = 32
	MaxSchemaNodes          = 4096
	MaxProperties           = 128
	MaxRequired             = 128
	MaxEnumValues           = 256
	MaxApplicatorBranches   = 32
	MaxDefinitions          = 64
	MaxSchemaReferences     = 512
	MaxReferenceResolutions = 4096
	MaxEvaluationSteps      = 100000
	MaxInstanceBytes        = 1048576
	MaxInstanceDepth        = 64
	MaxObjectProperties     = 1024
	MaxArrayElements        = 4096
	MaxStringBytes          = 262144
	MaxInstanceNodes        = 100000
)

var (
	ErrInvalidSchema       = errors.New("invalid JSON Schema document")
	ErrUnsupportedSchema   = errors.New("unsupported JSON Schema contract")
	ErrInvalidInstance     = errors.New("invalid JSON instance")
	ErrSchemaLimitExceeded = errors.New("JSON Schema resource limit exceeded")
	ErrEvaluationLimit     = errors.New("JSON Schema evaluation budget exhausted")
)

var supportedKeywords = map[string]bool{
	"$schema": true, "$ref": true, "$defs": true,
	"type": true, "enum": true, "const": true,
	"properties": true, "required": true, "additionalProperties": true,
	"minProperties": true, "maxProperties": true,
	"items": true, "minItems": true, "maxItems": true, "uniqueItems": true,
	"minLength": true, "maxLength": true,
	"minimum": true, "maximum": true, "exclusiveMinimum": true,
	"exclusiveMaximum": true, "multipleOf": true,
	"allOf": true, "anyOf": true, "oneOf": true, "not": true,
}

type Validator struct {
	root map[string]any
	defs map[string]any
}

type schemaBudget struct {
	nodes int
	refs  int
}

type evaluationBudget struct {
	steps int
	refs  int
}

// Compile validates and compiles one bounded Draft 2020-12 schema document.
// It intentionally supports only the POCS-0001 vocabulary and local $defs refs.
func Compile(document []byte) (*Validator, error) {
	if len(document) == 0 || len(document) > MaxDocumentBytes || !utf8.Valid(document) || bytes.HasPrefix(document, []byte{0xef, 0xbb, 0xbf}) {
		return nil, ErrInvalidSchema
	}
	value, nodes, err := decodeBoundedJSON(document, MaxSchemaDepth, MaxSchemaNodes, 0)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSchema, err)
	}
	root, ok := value.(map[string]any)
	if !ok {
		return nil, ErrUnsupportedSchema
	}
	if root["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
		return nil, ErrUnsupportedSchema
	}
	if _, ok := root["type"]; !ok {
		return nil, ErrUnsupportedSchema
	}
	defs, _ := root["$defs"].(map[string]any)
	if len(defs) > MaxDefinitions {
		return nil, ErrSchemaLimitExceeded
	}
	budget := schemaBudget{nodes: nodes}
	if err := inspectSchema(root, 1, &budget); err != nil {
		return nil, err
	}
	if err := validateReferences(root, defs); err != nil {
		return nil, err
	}
	return &Validator{root: root, defs: defs}, nil
}

// Validate checks one JSON instance. All parsing and evaluation limits are
// deterministic; unsupported or malformed input fails closed.
func (v *Validator) Validate(instance []byte) error {
	if v == nil || v.root == nil {
		return ErrUnsupportedSchema
	}
	value, _, err := decodeBoundedJSON(instance, MaxInstanceDepth, MaxInstanceNodes, MaxInstanceBytes)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInstance, err)
	}
	if err := inspectInstance(value, 1); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInstance, err)
	}
	budget := evaluationBudget{}
	if err := v.validate(v.root, value, &budget); err != nil {
		return err
	}
	return nil
}

func decodeBoundedJSON(data []byte, maxDepth, maxNodes int, maxBytes int) (any, int, error) {
	if len(data) == 0 || (maxBytes > 0 && len(data) > maxBytes) || !utf8.Valid(data) {
		return nil, 0, errors.New("empty, oversized, or invalid UTF-8 JSON")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	nodes := 0
	var parse func(int) (any, error)
	parse = func(depth int) (any, error) {
		if depth > maxDepth {
			return nil, errors.New("JSON nesting depth exceeded")
		}
		nodes++
		if nodes > maxNodes {
			return nil, errors.New("JSON node limit exceeded")
		}
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		d, ok := tok.(json.Delim)
		if !ok {
			return tok, nil
		}
		switch d {
		case '{':
			obj := make(map[string]any)
			for dec.More() {
				keyToken, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, errors.New("object key is not a string")
				}
				if _, exists := obj[key]; exists {
					return nil, fmt.Errorf("duplicate object key %q", key)
				}
				value, err := parse(depth + 1)
				if err != nil {
					return nil, err
				}
				obj[key] = value
				if len(obj) > MaxObjectProperties {
					return nil, errors.New("object property limit exceeded")
				}
			}
			end, err := dec.Token()
			if err != nil || end != json.Delim('}') {
				return nil, errors.New("unterminated object")
			}
			return obj, nil
		case '[':
			arr := make([]any, 0)
			for dec.More() {
				value, err := parse(depth + 1)
				if err != nil {
					return nil, err
				}
				arr = append(arr, value)
				if len(arr) > MaxArrayElements {
					return nil, errors.New("array element limit exceeded")
				}
			}
			end, err := dec.Token()
			if err != nil || end != json.Delim(']') {
				return nil, errors.New("unterminated array")
			}
			return arr, nil
		default:
			return nil, errors.New("unexpected JSON delimiter")
		}
	}
	value, err := parse(1)
	if err != nil {
		return nil, nodes, err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return nil, nodes, errors.New("multiple JSON values")
		}
		return nil, nodes, err
	}
	return value, nodes, nil
}

func inspectInstance(value any, depth int) error {
	if depth > MaxInstanceDepth {
		return errors.New("instance depth exceeded")
	}
	switch x := value.(type) {
	case string:
		if len([]byte(x)) > MaxStringBytes {
			return errors.New("instance string too large")
		}
	case map[string]any:
		if len(x) > MaxObjectProperties {
			return errors.New("too many object properties")
		}
		for _, child := range x {
			if err := inspectInstance(child, depth+1); err != nil {
				return err
			}
		}
	case []any:
		if len(x) > MaxArrayElements {
			return errors.New("too many array elements")
		}
		for _, child := range x {
			if err := inspectInstance(child, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func inspectSchema(s map[string]any, depth int, budget *schemaBudget) error {
	if depth > MaxSchemaDepth {
		return ErrSchemaLimitExceeded
	}

	for key, raw := range s {
		if !supportedKeywords[key] {
			return fmt.Errorf("%w: keyword %q", ErrUnsupportedSchema, key)
		}
		switch key {
		case "$schema":
			if raw != "https://json-schema.org/draft/2020-12/schema" {
				return ErrUnsupportedSchema
			}
		case "$ref":
			ref, ok := raw.(string)
			if !ok || !validLocalRef(ref) {
				return ErrUnsupportedSchema
			}
			budget.refs++
			if budget.refs > MaxSchemaReferences {
				return ErrSchemaLimitExceeded
			}
		case "$defs":
			defs, ok := raw.(map[string]any)
			if !ok || len(defs) > MaxDefinitions {
				return ErrInvalidSchema
			}
			for _, child := range defs {
				obj, ok := child.(map[string]any)
				if !ok {
					return ErrUnsupportedSchema
				}
				if err := inspectSchema(obj, depth+1, budget); err != nil {
					return err
				}
			}
		case "type":
			if !validType(raw) {
				return ErrUnsupportedSchema
			}
		case "enum":
			a, ok := raw.([]any)
			if !ok || len(a) == 0 || len(a) > MaxEnumValues {
				return ErrInvalidSchema
			}
		case "const":
			// Any JSON value is permitted.
		case "properties":
			props, ok := raw.(map[string]any)
			if !ok || len(props) > MaxProperties {
				return ErrInvalidSchema
			}
			for _, child := range props {
				obj, ok := child.(map[string]any)
				if !ok {
					return ErrUnsupportedSchema
				}
				if err := inspectSchema(obj, depth+1, budget); err != nil {
					return err
				}
			}
		case "required":
			a, ok := raw.([]any)
			if !ok || len(a) > MaxRequired {
				return ErrSchemaLimitExceeded
			}
			seen := map[string]bool{}
			for _, item := range a {
				name, ok := item.(string)
				if !ok || seen[name] {
					return ErrInvalidSchema
				}
				seen[name] = true
			}
		case "additionalProperties":
			if raw != false {
				return ErrUnsupportedSchema
			}
		case "items", "not":
			obj, ok := raw.(map[string]any)
			if !ok {
				return ErrUnsupportedSchema
			}
			if err := inspectSchema(obj, depth+1, budget); err != nil {
				return err
			}
		case "allOf", "anyOf", "oneOf":
			a, ok := raw.([]any)
			if !ok || len(a) == 0 || len(a) > MaxApplicatorBranches {
				return ErrSchemaLimitExceeded
			}
			for _, child := range a {
				obj, ok := child.(map[string]any)
				if !ok {
					return ErrUnsupportedSchema
				}
				if err := inspectSchema(obj, depth+1, budget); err != nil {
					return err
				}
			}
		case "minProperties", "maxProperties", "minItems", "maxItems", "minLength", "maxLength":
			if _, ok := nonNegativeInt(raw); !ok {
				return ErrInvalidSchema
			}
		case "uniqueItems":
			if _, ok := raw.(bool); !ok {
				return ErrInvalidSchema
			}
		case "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf":
			if _, ok := rational(raw); !ok {
				return ErrInvalidSchema
			}
			if key == "multipleOf" {
				n, _ := rational(raw)
				if n.Sign() <= 0 {
					return ErrInvalidSchema
				}
			}
		}
	}
	return nil
}

func validLocalRef(ref string) bool {
	if ref == "#" {
		return true
	}
	if !strings.HasPrefix(ref, "#/$defs/") {
		return false
	}
	name := strings.TrimPrefix(ref, "#/$defs/")
	return name != "" && !strings.Contains(name, "/") && !strings.Contains(name, "~") && !strings.Contains(name, "%")
}

func validType(v any) bool {
	valid := map[string]bool{"object": true, "array": true, "string": true, "number": true, "integer": true, "boolean": true, "null": true}
	if s, ok := v.(string); ok {
		return valid[s]
	}
	a, ok := v.([]any)
	if !ok || len(a) == 0 {
		return false
	}
	seen := map[string]bool{}
	for _, item := range a {
		s, ok := item.(string)
		if !ok || !valid[s] || seen[s] {
			return false
		}
		seen[s] = true
	}
	return true
}

func nonNegativeInt(v any) (int, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(string(n))
	return i, err == nil && i >= 0
}

func rational(v any) (*big.Rat, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return nil, false
	}
	r, ok := new(big.Rat).SetString(string(n))
	return r, ok
}

func (v *Validator) validate(s map[string]any, instance any, b *evaluationBudget) error {
	for keyword, raw := range s {
		if keyword == "$schema" || keyword == "$defs" {
			continue
		}
		b.steps++
		if b.steps > MaxEvaluationSteps {
			return ErrEvaluationLimit
		}
		switch keyword {
		case "$ref":
			b.refs++
			if b.refs > MaxReferenceResolutions {
				return ErrEvaluationLimit
			}
			ref := raw.(string)
			target, _ := v.root.(map[string]any)
			if ref != "#" {
				name := strings.TrimPrefix(ref, "#/$defs/")
				var ok bool
				target, ok = v.defs[name].(map[string]any)
				if !ok {
					return ErrInvalidSchema
				}
			}
			if err := v.validate(target, instance, b); err != nil {
				return err
			}
		case "type":
			if !matchesType(raw, instance) {
				return ErrInvalidInstance
			}
		case "enum":
			found := false
			for _, candidate := range raw.([]any) {
				if jsonEqual(candidate, instance) {
					found = true
					break
				}
			}
			if !found {
				return ErrInvalidInstance
			}
		case "const":
			if !jsonEqual(raw, instance) {
				return ErrInvalidInstance
			}
		case "properties":
			obj, ok := instance.(map[string]any)
			if !ok {
				continue
			}
			for name, child := range raw.(map[string]any) {
				if value, exists := obj[name]; exists {
					if err := v.validate(child.(map[string]any), value, b); err != nil {
						return err
					}
				}
			}
		case "required":
			obj, ok := instance.(map[string]any)
			if !ok {
				continue
			}
			for _, item := range raw.([]any) {
				if _, exists := obj[item.(string)]; !exists {
					return ErrInvalidInstance
				}
			}
		case "additionalProperties":
			// false is enforced against the declared properties when present.
			// This is handled in the properties pass below by checking unknown keys.
			obj, ok := instance.(map[string]any)
			if !ok {
				continue
			}
			props, _ := s["properties"].(map[string]any)
			for name := range obj {
				if _, exists := props[name]; !exists {
					return ErrInvalidInstance
				}
			}
		case "minProperties", "maxProperties":
			obj, ok := instance.(map[string]any)
			if !ok {
				continue
			}
			n, _ := nonNegativeInt(raw)
			if keyword == "minProperties" && len(obj) < n || keyword == "maxProperties" && len(obj) > n {
				return ErrInvalidInstance
			}
		case "items":
			arr, ok := instance.([]any)
			if !ok {
				continue
			}
			for _, item := range arr {
				if err := v.validate(raw.(map[string]any), item, b); err != nil {
					return err
				}
			}
		case "minItems", "maxItems":
			arr, ok := instance.([]any)
			if !ok {
				continue
			}
			n, _ := nonNegativeInt(raw)
			if keyword == "minItems" && len(arr) < n || keyword == "maxItems" && len(arr) > n {
				return ErrInvalidInstance
			}
		case "uniqueItems":
			if raw == true {
				arr, ok := instance.([]any)
				if ok {
					for i := range arr {
						for j := 0; j < i; j++ {
							if jsonEqual(arr[i], arr[j]) {
								return ErrInvalidInstance
							}
						}
					}
				}
			}
		case "minLength", "maxLength":
			str, ok := instance.(string)
			if !ok {
				continue
			}
			n, _ := nonNegativeInt(raw)
			length := len([]rune(str))
			if keyword == "minLength" && length < n || keyword == "maxLength" && length > n {
				return ErrInvalidInstance
			}
		case "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf":
			value, ok := rational(instance)
			if !ok {
				continue
			}
			limit, _ := rational(raw)
			cmp := value.Cmp(limit)
			switch keyword {
			case "minimum":
				if cmp < 0 {
					return ErrInvalidInstance
				}
			case "maximum":
				if cmp > 0 {
					return ErrInvalidInstance
				}
			case "exclusiveMinimum":
				if cmp <= 0 {
					return ErrInvalidInstance
				}
			case "exclusiveMaximum":
				if cmp >= 0 {
					return ErrInvalidInstance
				}
			case "multipleOf":
				if new(big.Rat).Quo(value, limit).IsInt() == false {
					return ErrInvalidInstance
				}
			}
		case "allOf", "anyOf", "oneOf":
			matches := 0
			for _, branch := range raw.([]any) {
				local := evaluationBudget{}
				err := v.validate(branch.(map[string]any), instance, &local)
				if err == nil {
					matches++
				} else if !errors.Is(err, ErrInvalidInstance) {
					return err
				}
				if errors.Is(err, ErrEvaluationLimit) {
					return err
				}
				b.steps += local.steps
				b.refs += local.refs
				if b.steps > MaxEvaluationSteps || b.refs > MaxReferenceResolutions {
					return ErrEvaluationLimit
				}
			}
			if keyword == "allOf" && matches != len(raw.([]any)) || keyword == "anyOf" && matches == 0 || keyword == "oneOf" && matches != 1 {
				return ErrInvalidInstance
			}
		case "not":
			local := evaluationBudget{}
			err := v.validate(raw.(map[string]any), instance, &local)
			if errors.Is(err, ErrEvaluationLimit) {
				return err
			}
			if err == nil {
				return ErrInvalidInstance
			}
			if !errors.Is(err, ErrInvalidInstance) {
				return err
			}
			b.steps += local.steps
			b.refs += local.refs
			if b.steps > MaxEvaluationSteps || b.refs > MaxReferenceResolutions {
				return ErrEvaluationLimit
			}
		}
	}
	return nil
}

func matchesType(expected, value any) bool {
	match := func(name string) bool {
		switch name {
		case "object":
			_, ok := value.(map[string]any)
			return ok
		case "array":
			_, ok := value.([]any)
			return ok
		case "string":
			_, ok := value.(string)
			return ok
		case "number":
			_, ok := value.(json.Number)
			return ok
		case "integer":
			n, ok := value.(json.Number)
			if !ok {
				return false
			}
			r, ok := new(big.Rat).SetString(string(n))
			return ok && r.IsInt()
		case "boolean":
			_, ok := value.(bool)
			return ok
		case "null":
			return value == nil
		}
		return false
	}
	if name, ok := expected.(string); ok {
		return match(name)
	}
	for _, item := range expected.([]any) {
		if match(item.(string)) {
			return true
		}
	}
	return false
}

func jsonEqual(a, b any) bool {
	if x, ok := rational(a); ok {
		y, ok := rational(b)
		return ok && x.Cmp(y) == 0
	}
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			other, exists := y[k]
			if !exists || !jsonEqual(v, other) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !jsonEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}

// SupportedKeywords returns a stable copy of the bounded keyword vocabulary.
func SupportedKeywords() []string {
	out := make([]string, 0, len(supportedKeywords))
	for key := range supportedKeywords {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func validateReferences(value any, defs map[string]any) error {
	var walk func(any) error
	walk = func(node any) error {
		switch current := node.(type) {
		case map[string]any:
			if raw, exists := current["$ref"]; exists {
				ref, ok := raw.(string)
				if !ok || !validLocalRef(ref) {
					return ErrUnsupportedSchema
				}
				if ref != "#" {
					name := strings.TrimPrefix(ref, "#/$defs/")
					if _, exists := defs[name]; !exists {
						return fmt.Errorf("%w: unresolved local reference %q", ErrInvalidSchema, ref)
					}
				}
			}
			for _, child := range current {
				if err := walk(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range current {
				if err := walk(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(value)
}
