package capability

import (
	"errors"
	"testing"
)

func TestValidateDeclarationSet(t *testing.T) {
	tests := []struct {
		name         string
		declarations []CapabilityDeclaration
		wantErr      bool
	}{
		{
			name: "empty set",
		},
		{
			name: "unique declarations",
			declarations: []CapabilityDeclaration{
				{ID: "com.example/document.summarize"},
				{ID: "org.example/text.extract"},
			},
		},
		{
			name: "duplicate identifiers",
			declarations: []CapabilityDeclaration{
				{ID: "com.example/document.summarize"},
				{ID: "com.example/document.summarize"},
			},
			wantErr: true,
		},
		{
			name: "invalid identifier",
			declarations: []CapabilityDeclaration{
				{ID: "Com.example/document.summarize"},
			},
			wantErr: true,
		},
		{
			name: "reserved namespace",
			declarations: []CapabilityDeclaration{
				{ID: "io.pride/text.generate"},
			},
			wantErr: true,
		},
		{
			name: "too many declarations",
			declarations: makeDeclarations(MaxCapabilityDeclarations + 1),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateDeclarationSet(test.declarations)
			if test.wantErr && !errors.Is(err, ErrInvalidDeclarationSet) {
				t.Fatalf("ValidateDeclarationSet() error = %v, want ErrInvalidDeclarationSet", err)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("ValidateDeclarationSet() error = %v", err)
			}
		})
	}
}

func makeDeclarations(count int) []CapabilityDeclaration {
	declarations := make([]CapabilityDeclaration, count)
	for i := range declarations {
		declarations[i].ID = "com.example/capability-" + testIndex(i)
	}
	return declarations
}

func testIndex(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var result [20]byte
	position := len(result)
	for value > 0 {
		position--
		result[position] = digits[value%10]
		value /= 10
	}
	return string(result[position:])
}
