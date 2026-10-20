package configuration

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestSQLiteStoreCreateSelectAndGetActive(t *testing.T) {
	ctx := context.Background()

	store, err := NewSQLiteStore(
		ctx,
		"file:configuration-test?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf(
			"NewSQLiteStore() failed: %v",
			err,
		)
	}

	defer store.Close()

	expected := AIConfiguration{
		Ref:  "config_test_openai",
		Name: "test-openai",
		Type: "openai",

		ImplementationRef: "openai.responses",

		ConnectionParameters: map[string]any{
			"model":    "test-model",
			"base_url": "https://example.test",
			"timeout":  float64(30),
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
			"unexpected implementation ref: got %q, want %q",
			actual.ImplementationRef,
			expected.ImplementationRef,
		)
	}

	if actual.CredentialRef != expected.CredentialRef {
		t.Fatalf(
			"unexpected credential ref: got %q, want %q",
			actual.CredentialRef,
			expected.CredentialRef,
		)
	}

	if !actual.Enabled {
		t.Fatal(
			"expected selected configuration to be enabled",
		)
	}

	if actual.ConnectionParameters["model"] != "test-model" {
		t.Fatalf(
			"unexpected model: got %v",
			actual.ConnectionParameters["model"],
		)
	}

	if actual.ConnectionParameters["base_url"] != "https://example.test" {
		t.Fatalf(
			"unexpected base_url: got %v",
			actual.ConnectionParameters["base_url"],
		)
	}

	if actual.ConnectionParameters["timeout"] != float64(30) {
		t.Fatalf(
			"unexpected timeout: got %v",
			actual.ConnectionParameters["timeout"],
		)
	}
}

func TestSQLiteStoreGetByRef(t *testing.T) {
	ctx := context.Background()

	store, err := NewSQLiteStore(
		ctx,
		"file:configuration-get-by-ref?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf(
			"NewSQLiteStore() failed: %v",
			err,
		)
	}

	defer store.Close()

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
}

func TestSQLiteStoreGetByRefRejectsUnknownReference(t *testing.T) {
	ctx := context.Background()

	store, err := NewSQLiteStore(
		ctx,
		"file:configuration-get-by-ref-unknown?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf(
			"NewSQLiteStore() failed: %v",
			err,
		)
	}

	defer store.Close()

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

	_, err = store.GetByRef(
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

func TestSQLiteStoreGetActiveNotFound(t *testing.T) {
	ctx := context.Background()

	store, err := NewSQLiteStore(
		ctx,
		"file:configuration-not-found?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf(
			"NewSQLiteStore() failed: %v",
			err,
		)
	}

	defer store.Close()

	_, err = store.GetActive(ctx)

	if !errors.Is(
		err,
		ErrConfigurationNotFound,
	) {
		t.Fatalf(
			"unexpected error: got %v, want %v",
			err,
			ErrConfigurationNotFound,
		)
	}
}

func TestSQLiteStoreSelectMaintainsSingleActiveConfiguration(
	t *testing.T,
) {
	ctx := context.Background()

	store, err := NewSQLiteStore(
		ctx,
		"file:configuration-select?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf(
			"NewSQLiteStore() failed: %v",
			err,
		)
	}

	defer store.Close()

	configurations := []AIConfiguration{
		{
			Ref:               "config_a",
			Name:              "configuration-a",
			Type:              "provider-a",
			ImplementationRef: "implementation.a",
			Enabled:           false,
		},
		{
			Ref:               "config_b",
			Name:              "configuration-b",
			Type:              "provider-b",
			ImplementationRef: "implementation.b",
			Enabled:           false,
		},
		{
			Ref:               "config_c",
			Name:              "configuration-c",
			Type:              "provider-c",
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
				"Create(%q) failed: %v",
				configuration.Ref,
				err,
			)
		}
	}

	if err := store.Select(
		ctx,
		"config_b",
	); err != nil {
		t.Fatalf(
			"Select(config_b) failed: %v",
			err,
		)
	}

	active, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"GetActive() failed: %v",
			err,
		)
	}

	if active.Ref != "config_b" {
		t.Fatalf(
			"unexpected active ref: got %q, want %q",
			active.Ref,
			"config_b",
		)
	}

	if !active.Enabled {
		t.Fatal(
			"selected configuration must be enabled",
		)
	}

	configA, err := store.GetByRef(
		ctx,
		"config_a",
	)
	if err != nil {
		t.Fatalf(
			"GetByRef(config_a) failed: %v",
			err,
		)
	}

	if configA.Enabled {
		t.Fatal(
			"config_a must be disabled",
		)
	}

	configC, err := store.GetByRef(
		ctx,
		"config_c",
	)
	if err != nil {
		t.Fatalf(
			"GetByRef(config_c) failed: %v",
			err,
		)
	}

	if configC.Enabled {
		t.Fatal(
			"config_c must be disabled",
		)
	}

	if err := store.Select(
		ctx,
		"config_c",
	); err != nil {
		t.Fatalf(
			"Select(config_c) failed: %v",
			err,
		)
	}

	active, err = store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"GetActive() after reselection failed: %v",
			err,
		)
	}

	if active.Ref != "config_c" {
		t.Fatalf(
			"unexpected active ref after reselection: got %q, want %q",
			active.Ref,
			"config_c",
		)
	}

	if !active.Enabled {
		t.Fatal(
			"newly selected configuration must be enabled",
		)
	}

	configB, err := store.GetByRef(
		ctx,
		"config_b",
	)
	if err != nil {
		t.Fatalf(
			"GetByRef(config_b) after reselection failed: %v",
			err,
		)
	}

	if configB.Enabled {
		t.Fatal(
			"previously selected config_b must be disabled",
		)
	}

	configA, err = store.GetByRef(
		ctx,
		"config_a",
	)
	if err != nil {
		t.Fatalf(
			"GetByRef(config_a) after reselection failed: %v",
			err,
		)
	}

	if configA.Enabled {
		t.Fatal(
			"config_a must remain disabled",
		)
	}
}

func TestSQLiteStoreSelectUnknownReference(
	t *testing.T,
) {
	ctx := context.Background()

	store, err := NewSQLiteStore(
		ctx,
		"file:configuration-select-unknown?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf(
			"NewSQLiteStore() failed: %v",
			err,
		)
	}

	defer store.Close()

	err = store.Select(
		ctx,
		"config_does_not_exist",
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

func TestSQLiteStoreDeleteSelectedConfigurationClearsActive(
	t *testing.T,
) {
	ctx := context.Background()

	store, err := NewSQLiteStore(
		ctx,
		"file:configuration-delete-selected?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf(
			"NewSQLiteStore() failed: %v",
			err,
		)
	}

	defer store.Close()

	configuration := AIConfiguration{
		Ref:               "config_selected",
		Name:              "selected",
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

	if err := store.Select(
		ctx,
		configuration.Ref,
	); err != nil {
		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}

	if err := store.Delete(
		ctx,
		configuration.Ref,
	); err != nil {
		t.Fatalf(
			"Delete() failed: %v",
			err,
		)
	}

	_, err = store.GetActive(ctx)

	if !errors.Is(
		err,
		ErrConfigurationNotFound,
	) {
		t.Fatalf(
			"unexpected active error: got %v, want %v",
			err,
			ErrConfigurationNotFound,
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
			"unexpected GetByRef() error: got %v, want %v",
			err,
			ErrConfigurationRefNotFound,
		)
	}
}

func TestSQLiteStorePersistsAcrossConnections(
	t *testing.T,
) {
	ctx := context.Background()

	databasePath := filepath.Join(
		t.TempDir(),
		"configuration.db",
	)

	dsn := databasePath

	first, err := NewSQLiteStore(
		ctx,
		dsn,
	)
	if err != nil {
		t.Fatalf(
			"first NewSQLiteStore() failed: %v",
			err,
		)
	}

	configuration := AIConfiguration{
		Ref:  "config_persistent",
		Name: "persistent",
		Type: "test",

		ImplementationRef: "test.implementation",

		ConnectionParameters: map[string]any{
			"model": "persistent-model",
		},

		CredentialRef: "cred_persistent",

		Enabled: false,
	}

	if err := first.Create(
		ctx,
		configuration,
	); err != nil {
		first.Close()

		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if err := first.Select(
		ctx,
		configuration.Ref,
	); err != nil {
		first.Close()

		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}

	if err := first.Close(); err != nil {
		t.Fatalf(
			"first Close() failed: %v",
			err,
		)
	}

	second, err := NewSQLiteStore(
		ctx,
		dsn,
	)
	if err != nil {
		t.Fatalf(
			"second NewSQLiteStore() failed: %v",
			err,
		)
	}

	defer second.Close()

	result, err := second.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"GetActive() after reopen failed: %v",
			err,
		)
	}

	if result.Ref != configuration.Ref {
		t.Fatalf(
			"unexpected persisted Ref: got %q, want %q",
			result.Ref,
			configuration.Ref,
		)
	}

	if result.Name != configuration.Name {
		t.Fatalf(
			"unexpected persisted Name: got %q, want %q",
			result.Name,
			configuration.Name,
		)
	}

	if result.Type != configuration.Type {
		t.Fatalf(
			"unexpected persisted Type: got %q, want %q",
			result.Type,
			configuration.Type,
		)
	}

	if result.ImplementationRef != configuration.ImplementationRef {
		t.Fatalf(
			"unexpected persisted ImplementationRef: got %q, want %q",
			result.ImplementationRef,
			configuration.ImplementationRef,
		)
	}

	if result.CredentialRef != configuration.CredentialRef {
		t.Fatalf(
			"unexpected persisted CredentialRef: got %q, want %q",
			result.CredentialRef,
			configuration.CredentialRef,
		)
	}

	if !result.Enabled {
		t.Fatal(
			"selected configuration must remain enabled after reopen",
		)
	}

	if result.ConnectionParameters["model"] !=
		configuration.ConnectionParameters["model"] {
		t.Fatalf(
			"unexpected persisted model: got %v, want %v",
			result.ConnectionParameters["model"],
			configuration.ConnectionParameters["model"],
		)
	}
}
