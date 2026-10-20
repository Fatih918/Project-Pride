package configuration

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"pride-ai-module/credential"
)

type testKeySource struct {
	key []byte
}

func (s testKeySource) GetKey(
	ctx context.Context,
) ([]byte, error) {
	return s.key, nil
}

func newTestCredentialService(
	t *testing.T,
	ctx context.Context,
) (*credential.Service, func()) {
	t.Helper()

	key := []byte(
		"01234567890123456789012345678901",
	)

	credentialStore,
		sqliteCredentialStore,
		err := credential.NewSQLiteEncryptedStore(
		ctx,
		"file:configuration-service-credential?mode=memory&cache=shared",
		testKeySource{
			key: key,
		},
	)
	if err != nil {
		t.Fatalf(
			"NewSQLiteEncryptedStore() failed: %v",
			err,
		)
	}

	credentialService := credential.NewService(
		credentialStore,
	)

	cleanup := func() {
		sqliteCredentialStore.Close()
	}

	return credentialService, cleanup
}

func TestConfigurationServiceCreateWithCredential(
	t *testing.T,
) {
	ctx := context.Background()

	credentialService, cleanup :=
		newTestCredentialService(t, ctx)
	defer cleanup()

	configurationStore := NewMemoryStore()

	service := NewConfigurationService(
		configurationStore,
		credentialService,
	)

	secret := []byte("real-test-api-key")

	result, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "test-provider",

			Type: "remote",

			ImplementationRef: "test.implementation",

			ConnectionParameters: map[string]any{
				"model": "test-model",
			},

			Secret: secret,
		},
	)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if result.Ref == "" {
		t.Fatal(
			"configuration reference must be generated",
		)
	}

	if result.Name != "test-provider" {
		t.Fatalf(
			"unexpected configuration name: got %q",
			result.Name,
		)
	}

	if result.Type != "remote" {
		t.Fatalf(
			"unexpected configuration type: got %q",
			result.Type,
		)
	}

	if result.ImplementationRef !=
		"test.implementation" {
		t.Fatalf(
			"unexpected implementation ref: got %q",
			result.ImplementationRef,
		)
	}

	if result.CredentialRef == "" {
		t.Fatal(
			"configuration credential reference must not be empty",
		)
	}

	if result.Enabled {
		t.Fatal(
			"new configuration must be disabled",
		)
	}

	if result.ConnectionParameters["model"] !=
		"test-model" {
		t.Fatalf(
			"unexpected model: got %v",
			result.ConnectionParameters["model"],
		)
	}

	stored, err := configurationStore.GetByRef(
		ctx,
		result.Ref,
	)
	if err != nil {
		t.Fatalf(
			"GetByRef() failed: %v",
			err,
		)
	}

	if stored.Ref != result.Ref {
		t.Fatalf(
			"stored ref mismatch: got %q, want %q",
			stored.Ref,
			result.Ref,
		)
	}

	if stored.Enabled {
		t.Fatal(
			"stored new configuration must be disabled",
		)
	}

	resolvedCredential, err :=
		credentialService.Get(
			ctx,
			result.CredentialRef,
		)
	if err != nil {
		t.Fatalf(
			"credential Get() failed: %v",
			err,
		)
	}

	if !bytes.Equal(
		resolvedCredential.Value,
		secret,
	) {
		t.Fatal(
			"stored credential does not match supplied secret",
		)
	}
}

func TestConfigurationServiceCreateDoesNotActivate(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	service := NewConfigurationService(
		store,
		nil,
	)

	result, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "test",

			Type: "remote",

			ImplementationRef: "test.implementation",
		},
	)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	_, err = store.GetActive(ctx)

	if err != ErrConfigurationNotFound {
		t.Fatalf(
			"Create() unexpectedly activated configuration: %v",
			err,
		)
	}

	stored, err := store.GetByRef(
		ctx,
		result.Ref,
	)
	if err != nil {
		t.Fatalf(
			"GetByRef() failed: %v",
			err,
		)
	}

	if stored.Ref != result.Ref {
		t.Fatalf(
			"unexpected stored configuration ref: got %q, want %q",
			stored.Ref,
			result.Ref,
		)
	}
}

func TestConfigurationServiceCreateRejectsEmptyName(
	t *testing.T,
) {
	ctx := context.Background()

	service := NewConfigurationService(
		NewMemoryStore(),
		nil,
	)

	_, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Type: "remote",

			ImplementationRef: "test.implementation",
		},
	)

	if err == nil {
		t.Fatal(
			"Create() expected empty-name validation error",
		)
	}
}

func TestConfigurationServiceCreateRejectsEmptyType(
	t *testing.T,
) {
	ctx := context.Background()

	service := NewConfigurationService(
		NewMemoryStore(),
		nil,
	)

	_, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "test",

			ImplementationRef: "test.implementation",
		},
	)

	if err == nil {
		t.Fatal(
			"Create() expected empty-type validation error",
		)
	}
}

func TestConfigurationServiceCreateRejectsEmptyImplementation(
	t *testing.T,
) {
	ctx := context.Background()

	service := NewConfigurationService(
		NewMemoryStore(),
		nil,
	)

	_, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "test",

			Type: "remote",
		},
	)

	if err == nil {
		t.Fatal(
			"Create() expected implementation validation error",
		)
	}
}

func TestConfigurationServiceCreateGeneratesUniqueReferences(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	service := NewConfigurationService(
		store,
		nil,
	)

	first, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "provider-one",

			Type: "remote",

			ImplementationRef: "implementation.one",

			ConnectionParameters: map[string]any{
				"model": "model-one",
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"first Create() failed: %v",
			err,
		)
	}

	second, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "provider-two",

			Type: "remote",

			ImplementationRef: "implementation.two",

			ConnectionParameters: map[string]any{
				"model": "model-two",
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"second Create() failed: %v",
			err,
		)
	}

	if first.Ref == "" {
		t.Fatal(
			"first configuration reference must not be empty",
		)
	}

	if second.Ref == "" {
		t.Fatal(
			"second configuration reference must not be empty",
		)
	}

	if first.Ref == second.Ref {
		t.Fatalf(
			"configuration references must be unique: got %q",
			first.Ref,
		)
	}

	firstStored, err := store.GetByRef(
		ctx,
		first.Ref,
	)
	if err != nil {
		t.Fatalf(
			"GetByRef(first) failed: %v",
			err,
		)
	}

	secondStored, err := store.GetByRef(
		ctx,
		second.Ref,
	)
	if err != nil {
		t.Fatalf(
			"GetByRef(second) failed: %v",
			err,
		)
	}

	if firstStored.Name != "provider-one" {
		t.Fatalf(
			"first configuration was overwritten: got %q",
			firstStored.Name,
		)
	}

	if secondStored.Name != "provider-two" {
		t.Fatalf(
			"second configuration was not stored correctly: got %q",
			secondStored.Name,
		)
	}
}

func TestConfigurationServiceCreateDoesNotReplaceActiveConfiguration(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	active := AIConfiguration{
		Ref: "existing-active",

		Name: "existing",

		Type: "remote",

		ImplementationRef: "existing.implementation",

		Enabled: true,
	}

	if err := store.Create(
		ctx,
		active,
	); err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if err := store.Select(
		ctx,
		active.Ref,
	); err != nil {
		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}
	service := NewConfigurationService(
		store,
		nil,
	)

	created, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "new-user-configuration",

			Type: "remote",

			ImplementationRef: "new.implementation",
		},
	)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if created.Enabled {
		t.Fatal(
			"new configuration must not be enabled",
		)
	}

	currentActive, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"GetActive() failed: %v",
			err,
		)
	}

	if currentActive.Ref != active.Ref {
		t.Fatalf(
			"Create() replaced active configuration: got %q, want %q",
			currentActive.Ref,
			active.Ref,
		)
	}
}

func TestConfigurationServiceCreateWithCredentialRef(
	t *testing.T,
) {
	ctx := context.Background()

	credentialService, cleanup :=
		newTestCredentialService(t, ctx)
	defer cleanup()

	configurationStore := NewMemoryStore()

	credentialRef, err := credentialService.Create(
		ctx,
		[]byte("existing-api-key"),
	)
	if err != nil {
		t.Fatalf(
			"credential Create() failed: %v",
			err,
		)
	}

	service := NewConfigurationService(
		configurationStore,
		credentialService,
	)

	result, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "existing-credential",

			Type: "remote",

			ImplementationRef: "test.implementation",

			CredentialRef: credentialRef,
		},
	)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if result.CredentialRef != credentialRef {
		t.Fatalf(
			"credential reference changed: got %q, want %q",
			result.CredentialRef,
			credentialRef,
		)
	}

	resolvedCredential, err :=
		credentialService.Get(
			ctx,
			credentialRef,
		)
	if err != nil {
		t.Fatalf(
			"credential Get() failed: %v",
			err,
		)
	}

	if !bytes.Equal(
		resolvedCredential.Value,
		[]byte("existing-api-key"),
	) {
		t.Fatal(
			"existing credential was modified",
		)
	}
}

func TestConfigurationServiceCreateRejectsCredentialRefAndSecret(
	t *testing.T,
) {
	ctx := context.Background()

	credentialService, cleanup :=
		newTestCredentialService(t, ctx)
	defer cleanup()

	service := NewConfigurationService(
		NewMemoryStore(),
		credentialService,
	)

	_, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "invalid",

			Type: "remote",

			ImplementationRef: "test.implementation",

			CredentialRef: "cred_existing",

			Secret: []byte("new-secret"),
		},
	)

	if err == nil {
		t.Fatal(
			"Create() expected credential ref and secret conflict error",
		)
	}
}

func TestConfigurationServiceCreateStoresOnlyCredentialReference(
	t *testing.T,
) {
	ctx := context.Background()

	credentialService, cleanup :=
		newTestCredentialService(t, ctx)
	defer cleanup()

	store := NewMemoryStore()

	service := NewConfigurationService(
		store,
		credentialService,
	)

	secret := []byte("super-secret-api-key")

	result, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "credential-boundary-test",

			Type: "remote",

			ImplementationRef: "test.implementation",

			Secret: secret,
		},
	)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if result.CredentialRef == "" {
		t.Fatal(
			"credential reference must not be empty",
		)
	}

	stored, err := store.GetByRef(
		ctx,
		result.Ref,
	)
	if err != nil {
		t.Fatalf(
			"GetByRef() failed: %v",
			err,
		)
	}

	if stored.CredentialRef != result.CredentialRef {
		t.Fatalf(
			"stored credential reference mismatch: got %q, want %q",
			stored.CredentialRef,
			result.CredentialRef,
		)
	}
}

func TestConfigurationServiceCreateDoesNotCreateCredentialWhenCredentialRefProvided(
	t *testing.T,
) {
	ctx := context.Background()

	credentialService, cleanup :=
		newTestCredentialService(t, ctx)
	defer cleanup()

	credentialStore := credentialService

	credentialRef, err := credentialStore.Create(
		ctx,
		[]byte("existing-secret"),
	)
	if err != nil {
		t.Fatalf(
			"credential Create() failed: %v",
			err,
		)
	}

	service := NewConfigurationService(
		NewMemoryStore(),
		credentialService,
	)

	result, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "reuse-credential",

			Type: "remote",

			ImplementationRef: "test.implementation",

			CredentialRef: credentialRef,
		},
	)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if result.CredentialRef != credentialRef {
		t.Fatalf(
			"unexpected credential reference: got %q, want %q",
			result.CredentialRef,
			credentialRef,
		)
	}
}

func TestConfigurationAndCredentialReferencesAreIndependent(
	t *testing.T,
) {
	ctx := context.Background()

	credentialService, cleanup :=
		newTestCredentialService(t, ctx)
	defer cleanup()

	configurationStore := NewMemoryStore()

	credentialRef, err := credentialService.Create(
		ctx,
		[]byte("independent-secret"),
	)
	if err != nil {
		t.Fatalf(
			"credential Create() failed: %v",
			err,
		)
	}

	configurationService := NewConfigurationService(
		configurationStore,
		credentialService,
	)

	configuration, err := configurationService.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "independent-reference-test",

			Type: "remote",

			ImplementationRef: "test.implementation",

			CredentialRef: credentialRef,
		},
	)
	if err != nil {
		t.Fatalf(
			"configuration Create() failed: %v",
			err,
		)
	}

	if configuration.Ref == "" {
		t.Fatal(
			"configuration reference must not be empty",
		)
	}

	if configuration.CredentialRef == "" {
		t.Fatal(
			"credential reference must not be empty",
		)
	}

	if configuration.Ref == configuration.CredentialRef {
		t.Fatalf(
			"configuration reference and credential reference must be different: %q",
			configuration.Ref,
		)
	}

	// Delete the configuration.
	if err := configurationStore.Delete(
		ctx,
		configuration.Ref,
	); err != nil {
		t.Fatalf(
			"configuration Delete() failed: %v",
			err,
		)
	}

	// The credential must still exist.
	resolvedCredential, err :=
		credentialService.Get(
			ctx,
			credentialRef,
		)
	if err != nil {
		t.Fatalf(
			"credential was deleted together with configuration: %v",
			err,
		)
	}

	if !bytes.Equal(
		resolvedCredential.Value,
		[]byte("independent-secret"),
	) {
		t.Fatal(
			"credential value changed after configuration deletion",
		)
	}
}
func TestDeletingCredentialDoesNotDeleteConfiguration(
	t *testing.T,
) {
	ctx := context.Background()

	credentialService, cleanup :=
		newTestCredentialService(t, ctx)
	defer cleanup()

	configurationStore := NewMemoryStore()

	credentialRef, err := credentialService.Create(
		ctx,
		[]byte("credential-to-delete"),
	)
	if err != nil {
		t.Fatalf(
			"credential Create() failed: %v",
			err,
		)
	}

	configurationService := NewConfigurationService(
		configurationStore,
		credentialService,
	)

	configuration, err := configurationService.Create(
		ctx,
		CreateConfigurationRequest{
			Name: "configuration-survives-credential-delete",

			Type: "remote",

			ImplementationRef: "test.implementation",

			CredentialRef: credentialRef,
		},
	)
	if err != nil {
		t.Fatalf(
			"configuration Create() failed: %v",
			err,
		)
	}

	// Delete the credential independently.
	if err := credentialService.Delete(
		ctx,
		credentialRef,
	); err != nil {
		t.Fatalf(
			"credential Delete() failed: %v",
			err,
		)
	}

	// Credential should no longer resolve.
	_, err = credentialService.Get(
		ctx,
		credentialRef,
	)
	if err == nil {
		t.Fatal(
			"deleted credential should not be resolvable",
		)
	}

	// Configuration must still exist.
	storedConfiguration, err :=
		configurationStore.GetByRef(
			ctx,
			configuration.Ref,
		)
	if err != nil {
		t.Fatalf(
			"configuration was deleted together with credential: %v",
			err,
		)
	}

	if storedConfiguration.Ref != configuration.Ref {
		t.Fatalf(
			"configuration reference changed: got %q, want %q",
			storedConfiguration.Ref,
			configuration.Ref,
		)
	}

	if storedConfiguration.CredentialRef != credentialRef {
		t.Fatalf(
			"configuration credential reference changed: got %q, want %q",
			storedConfiguration.CredentialRef,
			credentialRef,
		)
	}
}

func TestConfigurationServiceSelect(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	service := NewConfigurationService(
		store,
		nil,
	)

	configA, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name:              "Gemini",
			Type:              "text",
			ImplementationRef: "gemini",
		},
	)
	if err != nil {
		t.Fatalf("create config A: %v", err)
	}

	configB, err := service.Create(
		ctx,
		CreateConfigurationRequest{
			Name:              "OpenAI",
			Type:              "text",
			ImplementationRef: "openai",
		},
	)
	if err != nil {
		t.Fatalf("create config B: %v", err)
	}

	selected, err := service.Select(
		ctx,
		configA.Ref,
	)
	if err != nil {
		t.Fatalf("select config A: %v", err)
	}

	if selected.Ref != configA.Ref {
		t.Fatalf(
			"expected selected ref %q, got %q",
			configA.Ref,
			selected.Ref,
		)
	}

	if !selected.Enabled {
		t.Fatal("expected selected configuration to be enabled")
	}

	active, err := store.GetActive(ctx)
	if err != nil {
		t.Fatalf("get active configuration: %v", err)
	}

	if active.Ref != configA.Ref {
		t.Fatalf(
			"expected active ref %q, got %q",
			configA.Ref,
			active.Ref,
		)
	}

	selected, err = service.Select(
		ctx,
		configB.Ref,
	)
	if err != nil {
		t.Fatalf("select config B: %v", err)
	}

	if selected.Ref != configB.Ref {
		t.Fatalf(
			"expected selected ref %q, got %q",
			configB.Ref,
			selected.Ref,
		)
	}

	if !selected.Enabled {
		t.Fatal("expected selected configuration B to be enabled")
	}

	configA, err = store.GetByRef(ctx, configA.Ref)
	if err != nil {
		t.Fatalf("get config A after selecting B: %v", err)
	}

	if configA.Enabled {
		t.Fatal("expected previously selected config A to be disabled")
	}

	active, err = store.GetActive(ctx)
	if err != nil {
		t.Fatalf(
			"get active configuration after selecting B: %v",
			err,
		)
	}

	if active.Ref != configB.Ref {
		t.Fatalf(
			"expected active ref %q, got %q",
			configB.Ref,
			active.Ref,
		)
	}
}

func TestConfigurationServiceSelectEmptyReference(
	t *testing.T,
) {
	ctx := context.Background()

	store := NewMemoryStore()

	service := NewConfigurationService(
		store,
		nil,
	)

	_, err := service.Select(ctx, "")

	if !errors.Is(err, ErrConfigurationRefNotFound) {
		t.Fatalf(
			"expected ErrConfigurationRefNotFound, got %v",
			err,
		)
	}
}
