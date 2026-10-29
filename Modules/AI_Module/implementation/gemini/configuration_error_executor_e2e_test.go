package gemini

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	prideerrors "pride-ai-module/errors"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/selection"
)

func TestGeminiInvalidConfigurationThroughExecutor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configurationStore, err := configuration.NewSQLiteStore(
		ctx,
		filepath.Join(t.TempDir(), "configuration.db"),
	)
	if err != nil {
		t.Fatalf("create SQLite configuration store: %v", err)
	}
	t.Cleanup(func() {
		if err := configurationStore.Close(); err != nil {
			t.Errorf("close SQLite configuration store: %v", err)
		}
	})

	encryptedCredentialStore, credentialDB, err := credential.NewSQLiteEncryptedStore(
		ctx,
		filepath.Join(t.TempDir(), "credentials.db"),
		fullE2EKeySource{},
	)
	if err != nil {
		t.Fatalf("create encrypted SQLite credential store: %v", err)
	}
	t.Cleanup(func() {
		if err := credentialDB.Close(); err != nil {
			t.Errorf("close SQLite credential database: %v", err)
		}
	})

	credentialService := credential.NewService(encryptedCredentialStore)
	credentialRef, err := credentialService.Create(ctx, []byte("gemini-configuration-test-key"))
	if err != nil {
		t.Fatalf("store test credential: %v", err)
	}

	const configurationRef = "gemini.invalid-configuration-e2e.configuration"
	if err := configurationStore.Create(ctx, configuration.AIConfiguration{
		Ref:               configurationRef,
		Name:              "Gemini Invalid Configuration E2E",
		Type:              "gemini",
		ImplementationRef: "gemini",
		CredentialRef:     credentialRef,
		Enabled:           true,
		// The model parameter is intentionally omitted. The real provider
		// factory must reject this before constructing a Gemini API client.
		ConnectionParameters: map[string]any{},
	}); err != nil {
		t.Fatalf("create AI configuration: %v", err)
	}
	if err := configurationStore.Select(ctx, configurationRef); err != nil {
		t.Fatalf("select AI configuration: %v", err)
	}

	selectionService := selection.NewService(selection.NewMemoryStore())
	if err := selectionService.Select(ctx, configurationRef); err != nil {
		t.Fatalf("select configuration reference for executor: %v", err)
	}

	adapter, err := NewAdapterWithFactory(NewRealProviderFactory())
	if err != nil {
		t.Fatalf("create Gemini adapter: %v", err)
	}
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Errorf("close Gemini adapter: %v", err)
		}
	})

	implementationResolver := implementation.NewResolver()
	if err := implementationResolver.Register("gemini", adapter); err != nil {
		t.Fatalf("register Gemini adapter: %v", err)
	}

	executor := execution.NewExecutor(
		configurationStore,
		selectionService,
		implementationResolver,
		credential.NewResolver(credentialService),
	)

	input, err := anypb.New(&aiv1.TextGenerateInput{Prompt: "Say hello"})
	if err != nil {
		t.Fatalf("encode TextGenerateInput: %v", err)
	}
	const requestID = "gemini-invalid-configuration-e2e"
	result, err := executor.Execute(ctx, executioncontract.AIExecutionRequest{
		RequestID:  requestID,
		Capability: CapabilityTextGenerate,
		Input:      input,
	})
	if err == nil {
		t.Fatal("expected configuration error, got successful result")
	}
	if result.Output != nil {
		t.Fatal("failed execution must not return output")
	}

	var prideErr *prideerrors.Error
	if !errors.As(err, &prideErr) {
		t.Fatalf("expected normalized PRIDE error, got %T", err)
	}
	if prideErr.Code != prideerrors.CodeConfigurationError {
		t.Fatalf("error code = %q, want %q", prideErr.Code, prideerrors.CodeConfigurationError)
	}
	if prideErr.RequestID != requestID {
		t.Fatalf("request ID = %q, want %q", prideErr.RequestID, requestID)
	}
	if prideErr.Cause == nil {
		t.Fatal("expected internal configuration cause to remain available")
	}
	if !strings.Contains(prideErr.Cause.Error(), ConnectionParameterModel) {
		t.Fatalf("internal cause %q does not identify the invalid model configuration", prideErr.Cause)
	}
	if !strings.Contains(err.Error(), "Gemini provider resolution failed") {
		t.Fatalf("public error = %q, want normalized provider-resolution message", err)
	}
	if strings.Contains(err.Error(), "connection parameter") ||
		strings.Contains(err.Error(), "gemini-configuration-test-key") {
		t.Fatal("public error string exposed internal configuration or credential details")
	}
}
