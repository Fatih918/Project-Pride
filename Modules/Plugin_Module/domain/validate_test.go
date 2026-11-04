package domain

import (
	"errors"
	"testing"
)

func validPlugin() Plugin {
	id := PluginID("com.example.notes")
	return Plugin{
		ID: id,
		Manifest: Manifest{
			PluginID: id, Name: "Notes", Version: "1.0.0",
			Description: "Example notes plugin", Author: "PRIDE",
			License: "MIT", EntryPoint: "main.go", MinimumPRIDEVersion: "1.0.0",
		},
		Type: PluginTypeActive, State: PluginStateInstalled,
		Protected: ProtectedMetadata{
			Capabilities: []Capability{{ID: "notes.read", Name: "Read notes"}},
			PermissionIDs: []string{"notes.read"},
		},
	}
}

func TestPluginValidateAcceptsValidRecord(t *testing.T) {
	if err := validPlugin().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestPluginValidateRejectsMismatchedIdentity(t *testing.T) {
	plugin := validPlugin()
	plugin.Manifest.PluginID = "com.example.other"
	if err := plugin.Validate(); !errors.Is(err, ErrInvalidPlugin) {
		t.Fatalf("Validate() error = %v, want ErrInvalidPlugin", err)
	}
}

func TestPluginValidateRejectsDuplicateCapability(t *testing.T) {
	plugin := validPlugin()
	plugin.Protected.Capabilities = append(plugin.Protected.Capabilities, plugin.Protected.Capabilities[0])
	if err := plugin.Validate(); !errors.Is(err, ErrInvalidPlugin) {
		t.Fatalf("Validate() error = %v, want ErrInvalidPlugin", err)
	}
}

func TestPluginCloneCopiesMutableFields(t *testing.T) {
	original := validPlugin()
	clone := original.Clone()
	clone.Manifest.Keywords = append(clone.Manifest.Keywords, "changed")
	clone.Protected.Capabilities[0].Name = "changed"
	clone.Protected.PermissionIDs[0] = "changed"
	clone.Protected.Attributes = map[string]any{"key": "changed"}

	if len(original.Manifest.Keywords) != 0 ||
		original.Protected.Capabilities[0].Name != "Read notes" ||
		original.Protected.PermissionIDs[0] != "notes.read" ||
		original.Protected.Attributes != nil {
		t.Fatal("Clone() shares mutable fields with the original")
	}
}
