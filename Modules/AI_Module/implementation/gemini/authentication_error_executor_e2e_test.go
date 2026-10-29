package gemini

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	prideerrors "pride-ai-module/errors"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/provider"
	"pride-ai-module/selection"
)

func TestGeminiAuthenticationFailureThroughExecutor(t *testing.T) {
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
	credentialRef, err := credentialService.Create(ctx, []byte("invalid-gemini-test-key"))
	if err != nil {
		t.Fatalf("store test credential: %v", err)
	}

	const configurationRef = "gemini.authentication-e2e.configuration"
	const model = "gemini-test-model"
	if err := configurationStore.Create(ctx, configuration.AIConfiguration{
		Ref:               configurationRef,
		Name:              "Gemini Authentication Error E2E",
		Type:              "gemini",
		ImplementationRef: "gemini",
		CredentialRef:     credentialRef,
		Enabled:           true,
		ConnectionParameters: map[string]any{
			ConnectionParameterModel: model,
		},
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

	const providerMessage = "API key is invalid: api-key=DO_NOT_EXPOSE"
	fakeClient := &fakeContentGenerator{
		err: genai.APIError{
			Code:    401,
			Status:  "UNAUTHENTICATED",
			Message: providerMessage,
		},
	}
	adapter, err := NewAdapterWithFactory(func(
		factoryContext context.Context,
		executionContext executioncontract.AIExecutionContext,
	) (provider.Provider, error) {
		if factoryContext == nil {
			t.Fatal("provider factory received nil context")
		}
		if string(executionContext.Credential) != "invalid-gemini-test-key" {
			t.Fatal("provider factory did not receive the decrypted test credential")
		}
		return &RealProvider{client: fakeClient, model: model}, nil
	})
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
	const requestID = "gemini-authentication-error-e2e"
	result, err := executor.Execute(ctx, executioncontract.AIExecutionRequest{
		RequestID:  requestID,
		Capability: CapabilityTextGenerate,
		Input:      input,
	})
	if err == nil {
		t.Fatal("expected authentication error, got successful result")
	}
	if result.Output != nil {
		t.Fatal("failed execution must not return output")
	}

	var prideErr *prideerrors.Error
	if !errors.As(err, &prideErr) {
		t.Fatalf("expected normalized PRIDE error, got %T", err)
	}
	if prideErr.Code != prideerrors.CodeAuthenticationError {
		t.Fatalf("error code = %q, want %q", prideErr.Code, prideerrors.CodeAuthenticationError)
	}
	if prideErr.RequestID != requestID {
		t.Fatalf("request ID = %q, want %q", prideErr.RequestID, requestID)
	}
	if prideErr.Cause == nil {
		t.Fatal("expected internal provider cause to remain available")
	}
	if strings.Contains(err.Error(), providerMessage) || strings.Contains(err.Error(), "DO_NOT_EXPOSE") {
		t.Fatal("public error string exposed provider details")
	}
	if fakeClient.gotModel != model {
		t.Fatalf("provider model = %q, want %q", fakeClient.gotModel, model)
	}
	if len(fakeClient.gotContents) != 1 ||
		len(fakeClient.gotContents[0].Parts) != 1 ||
		fakeClient.gotContents[0].Parts[0].Text != "Say hello" {
		t.Fatal("provider did not receive the expected prompt")
	}
}
