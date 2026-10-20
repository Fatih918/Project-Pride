package configuration

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryStoreGetActiveNotFound(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	_, err := store.GetActive(ctx)

	if !errors.Is(err, ErrConfigurationNotFound) {
		t.Fatalf(
			"unexpected error: got %v, want %v",
			err,
			ErrConfigurationNotFound,
		)
	}
}

func TestMemoryStoreCreateSelectAndGetActive(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	expected := AIConfiguration{
		Ref:  "config_test_openai",
		Name: "test-openai",
		Type: "openai",

		ImplementationRef: "openai.responses",

		ConnectionParameters: map[string]any{
			"model":    "test-model",
			"base_url": "https://example.test",
		},

		CredentialRef: "cred_test_001",

		Enabled: false,
	}

	if err := store.Create(
		ctx,
		expected,
	); err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if err := store.Select(
		ctx,
		expected.Ref,
	); err != nil {
		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}

	actual, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"GetActive() failed: %v",
			err,
		)
	}

	if actual.Ref != expected.Ref {
		t.Fatalf(
			"unexpected Ref: got %q, want %q",
			actual.Ref,
			expected.Ref,
		)
	}

	if actual.Name != expected.Name {
		t.Fatalf(
			"unexpected Name: got %q, want %q",
			actual.Name,
			expected.Name,
		)
	}

	if actual.Type != expected.Type {
		t.Fatalf(
			"unexpected Type: got %q, want %q",
			actual.Type,
			expected.Type,
		)
	}

	if actual.ImplementationRef != expected.ImplementationRef {
		t.Fatalf(
			"unexpected ImplementationRef: got %q, want %q",
			actual.ImplementationRef,
			expected.ImplementationRef,
		)
	}

	if actual.CredentialRef != expected.CredentialRef {
		t.Fatalf(
			"unexpected CredentialRef: got %q, want %q",
			actual.CredentialRef,
			expected.CredentialRef,
		)
	}

	if !actual.Enabled {
		t.Fatal(
			"selected configuration must be enabled",
		)
	}

	if actual.ConnectionParameters["model"] !=
		expected.ConnectionParameters["model"] {
		t.Fatalf(
			"unexpected model parameter: got %v, want %v",
			actual.ConnectionParameters["model"],
			expected.ConnectionParameters["model"],
		)
	}
}

func TestMemoryStoreGetByRef(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	expected := AIConfiguration{
		Ref:               "config_openai",
		Name:              "openai",
		Type:              "openai",
		ImplementationRef: "openai.responses",
		Enabled:           false,
	}

	if err := store.Create(
		ctx,
		expected,
	); err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	actual, err := store.GetByRef(
		ctx,
		"config_openai",
	)
	if err != nil {
		t.Fatalf(
			"GetByRef() failed: %v",
			err,
		)
	}

	if actual.Ref != expected.Ref {
		t.Fatalf(
			"unexpected Ref: got %q, want %q",
			actual.Ref,
			expected.Ref,
		)
	}

	if actual.Name != expected.Name {
		t.Fatalf(
			"unexpected Name: got %q, want %q",
			actual.Name,
			expected.Name,
		)
	}

	if actual.Enabled {
		t.Fatal(
			"newly created configuration must be disabled",
		)
	}
}

func TestMemoryStoreGetByRefRejectsUnknownReference(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	configuration := AIConfiguration{
		Ref:               "config_openai",
		Name:              "openai",
		Type:              "openai",
		ImplementationRef: "openai.responses",
		Enabled:           false,
	}

	if err := store.Create(
		ctx,
		configuration,
	); err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	_, err := store.GetByRef(
		ctx,
		"config_anthropic",
	)

	if !errors.Is(
		err,
		ErrConfigurationRefNotFound,
	) {
		t.Fatalf(
			"unexpected error: got %v, want %v",
			err,
			ErrConfigurationRefNotFound,
		)
	}
}

func TestMemoryStoreDoesNotExposeInternalMap(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	configuration := AIConfiguration{
		Ref:               "config_test",
		Name:              "test",
		Type:              "test",
		ImplementationRef: "test.implementation",

		ConnectionParameters: map[string]any{
			"model": "original",
		},

		Enabled: false,
	}

	if err := store.Create(
		ctx,
		configuration,
	); err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if err := store.Select(
		ctx,
		configuration.Ref,
	); err != nil {
		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}

	result, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"GetActive() failed: %v",
			err,
		)
	}

	result.ConnectionParameters["model"] = "modified"

	again, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"second GetActive() failed: %v",
			err,
		)
	}

	if again.ConnectionParameters["model"] != "original" {
		t.Fatal(
			"GetActive() exposed internal connection parameters map",
		)
	}
}

func TestMemoryStoreCreateDoesNotActivateConfiguration(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	configuration := AIConfiguration{
		Ref:               "config_created",
		Name:              "created",
		Type:              "test",
		ImplementationRef: "test.implementation",
		Enabled:           false,
	}

	if err := store.Create(
		ctx,
		configuration,
	); err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	stored, err := store.GetByRef(
		ctx,
		configuration.Ref,
	)
	if err != nil {
		t.Fatalf(
			"GetByRef() failed: %v",
			err,
		)
	}

	if stored.Enabled {
		t.Fatal(
			"Create() must not enable configuration",
		)
	}

	_, err = store.GetActive(ctx)

	if !errors.Is(
		err,
		ErrConfigurationNotFound,
	) {
		t.Fatalf(
			"Create() unexpectedly created active configuration: %v",
			err,
		)
	}
}

func TestMemoryStoreSelect(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	configA := AIConfiguration{
		Ref:               "config-a",
		Name:              "Configuration A",
		Type:              "text",
		ImplementationRef: "test-a",
		Enabled:           false,
	}

	configB := AIConfiguration{
		Ref:               "config-b",
		Name:              "Configuration B",
		Type:              "text",
		ImplementationRef: "test-b",
		Enabled:           false,
	}

	if err := store.Create(
		ctx,
		configA,
	); err != nil {
		t.Fatalf(
			"create config A: %v",
			err,
		)
	}

	if err := store.Create(
		ctx,
		configB,
	); err != nil {
		t.Fatalf(
			"create config B: %v",
			err,
		)
	}

	if err := store.Select(
		ctx,
		"config-a",
	); err != nil {
		t.Fatalf(
			"select config A: %v",
			err,
		)
	}

	gotA, err := store.GetByRef(
		ctx,
		"config-a",
	)
	if err != nil {
		t.Fatalf(
			"get config A: %v",
			err,
		)
	}

	if !gotA.Enabled {
		t.Fatal(
			"expected config A to be enabled",
		)
	}

	active, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"get active configuration: %v",
			err,
		)
	}

	if active.Ref != "config-a" {
		t.Fatalf(
			"expected config-a to be active, got %q",
			active.Ref,
		)
	}

	if err := store.Select(
		ctx,
		"config-b",
	); err != nil {
		t.Fatalf(
			"select config B: %v",
			err,
		)
	}

	gotA, err = store.GetByRef(
		ctx,
		"config-a",
	)
	if err != nil {
		t.Fatalf(
			"get config A after selecting B: %v",
			err,
		)
	}

	if gotA.Enabled {
		t.Fatal(
			"expected config A to be disabled",
		)
	}

	gotB, err := store.GetByRef(
		ctx,
		"config-b",
	)
	if err != nil {
		t.Fatalf(
			"get config B: %v",
			err,
		)
	}

	if !gotB.Enabled {
		t.Fatal(
			"expected config B to be enabled",
		)
	}

	active, err = store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"get active configuration after selecting B: %v",
			err,
		)
	}

	if active.Ref != "config-b" {
		t.Fatalf(
			"expected config-b to be active, got %q",
			active.Ref,
		)
	}
}

func TestMemoryStoreSelectUnknownReference(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	err := store.Select(
		ctx,
		"does-not-exist",
	)

	if !errors.Is(
		err,
		ErrConfigurationRefNotFound,
	) {
		t.Fatalf(
			"expected ErrConfigurationRefNotFound, got %v",
			err,
		)
	}
}

func TestMemoryStoreSelectMaintainsSingleActiveConfiguration(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	configurations := []AIConfiguration{
		{
			Ref:               "config-a",
			Name:              "Configuration A",
			Type:              "text",
			ImplementationRef: "implementation.a",
			Enabled:           false,
		},
		{
			Ref:               "config-b",
			Name:              "Configuration B",
			Type:              "text",
			ImplementationRef: "implementation.b",
			Enabled:           false,
		},
		{
			Ref:               "config-c",
			Name:              "Configuration C",
			Type:              "text",
			ImplementationRef: "implementation.c",
			Enabled:           false,
		},
	}

	for _, configuration := range configurations {
		if err := store.Create(
			ctx,
			configuration,
		); err != nil {
			t.Fatalf(
				"create configuration %q: %v",
				configuration.Ref,
				err,
			)
		}
	}

	if err := store.Select(
		ctx,
		"config-b",
	); err != nil {
		t.Fatalf(
			"select config-b: %v",
			err,
		)
	}

	active, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"get active configuration: %v",
			err,
		)
	}

	if active.Ref != "config-b" {
		t.Fatalf(
			"expected active config-b, got %q",
			active.Ref,
		)
	}

	if !active.Enabled {
		t.Fatal(
			"selected configuration must be enabled",
		)
	}

	for _, ref := range []string{
		"config-a",
		"config-c",
	} {
		configuration, err := store.GetByRef(
			ctx,
			ref,
		)

		if err != nil {
			t.Fatalf(
				"get configuration %q: %v",
				ref,
				err,
			)
		}

		if configuration.Enabled {
			t.Fatalf(
				"configuration %q must be disabled",
				ref,
			)
		}
	}

	if err := store.Select(
		ctx,
		"config-c",
	); err != nil {
		t.Fatalf(
			"select config-c: %v",
			err,
		)
	}

	active, err = store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"get active configuration after reselection: %v",
			err,
		)
	}

	if active.Ref != "config-c" {
		t.Fatalf(
			"expected active config-c, got %q",
			active.Ref,
		)
	}

	if !active.Enabled {
		t.Fatal(
			"newly selected configuration must be enabled",
		)
	}

	previous, err := store.GetByRef(
		ctx,
		"config-b",
	)
	if err != nil {
		t.Fatalf(
			"get previous configuration: %v",
			err,
		)
	}

	if previous.Enabled {
		t.Fatal(
			"previously selected configuration must be disabled",
		)
	}
}

func TestMemoryStoreDeleteSelectedConfigurationClearsActive(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	configuration := AIConfiguration{
		Ref:               "config-selected",
		Name:              "Selected Configuration",
		Type:              "text",
		ImplementationRef: "implementation.test",
		Enabled:           false,
	}

	if err := store.Create(
		ctx,
		configuration,
	); err != nil {
		t.Fatalf(
			"create configuration: %v",
			err,
		)
	}

	if err := store.Select(
		ctx,
		configuration.Ref,
	); err != nil {
		t.Fatalf(
			"select configuration: %v",
			err,
		)
	}

	if err := store.Delete(
		ctx,
		configuration.Ref,
	); err != nil {
		t.Fatalf(
			"delete selected configuration: %v",
			err,
		)
	}

	_, err := store.GetActive(ctx)

	if !errors.Is(
		err,
		ErrConfigurationNotFound,
	) {
		t.Fatalf(
			"expected ErrConfigurationNotFound after deleting active configuration, got %v",
			err,
		)
	}

	_, err = store.GetByRef(
		ctx,
		configuration.Ref,
	)

	if !errors.Is(
		err,
		ErrConfigurationRefNotFound,
	) {
		t.Fatalf(
			"expected ErrConfigurationRefNotFound after deletion, got %v",
			err,
		)
	}
}

func TestMemoryStoreRejectsInvalidConfiguration(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	err := store.Create(
		ctx,
		AIConfiguration{
			Name: "invalid",
		},
	)

	if err == nil {
		t.Fatal(
			"Create() expected validation error",
		)
	}

	_, err = store.GetActive(ctx)

	if !errors.Is(
		err,
		ErrConfigurationNotFound,
	) {
		t.Fatalf(
			"invalid configuration was stored: got %v, want %v",
			err,
			ErrConfigurationNotFound,
		)
	}
}

func TestMemoryStoreDeleteUnknownReference(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	err := store.Delete(
		ctx,
		"does-not-exist",
	)

	if !errors.Is(
		err,
		ErrConfigurationRefNotFound,
	) {
		t.Fatalf(
			"expected ErrConfigurationRefNotFound, got %v",
			err,
		)
	}
}
