package gemini

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	"pride-ai-module/execution"
	prideerrors "pride-ai-module/errors"
	"pride-ai-module/implementation"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/selection"
)

// TestGeminiLiveAPIThroughExecutor is an opt-in integration test. It makes a
// real request to Google Gemini only when GEMINI_API_KEY and GEMINI_MODEL are
// configured in the test process. Never print the API key or persist it in
// source control.
func TestGeminiLiveAPIThroughExecutor(t *testing.T) {
	apiKey := strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	model := strings.TrimSpace(os.Getenv("GEMINI_MODEL"))
	if apiKey == "" || model == "" {
		t.Skip("set GEMINI_API_KEY and GEMINI_MODEL to run the live Gemini API test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
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
	credentialRef, err := credentialService.Create(ctx, []byte(apiKey))
	if err != nil {
		t.Fatalf("store Gemini API credential: %v", err)
	}

	// Confirm the persisted credential payload is not the plaintext API key.
	storedCredential, err := credentialDB.Get(ctx, credentialRef)
	if err != nil {
		t.Fatalf("read encrypted credential record: %v", err)
	}
	if strings.Contains(string(storedCredential.Value), apiKey) {
		t.Fatal("credential database contains the plaintext API key")
	}

	const configurationRef = "gemini.live-api-test.configuration"
	configurationValue := configuration.AIConfiguration{
		Ref:               configurationRef,
		Name:              "Gemini Live API Test",
		Type:              "gemini",
		ImplementationRef: "gemini",
		CredentialRef:     credentialRef,
		Enabled:           true,
		ConnectionParameters: map[string]any{
			ConnectionParameterModel: model,
		},
	}
	if err := configurationStore.Create(ctx, configurationValue); err != nil {
		t.Fatalf("create AI configuration: %v", err)
	}
	if err := configurationStore.Select(ctx, configurationRef); err != nil {
		t.Fatalf("select AI configuration: %v", err)
	}

	selectionService := selection.NewService(selection.NewMemoryStore())
	if err := selectionService.Select(ctx, configurationRef); err != nil {
		t.Fatalf("select configuration for executor: %v", err)
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

	const prompt = "Reply with a short greeting and identify yourself as Gemini."
	input, err := anypb.New(&aiv1.TextGenerateInput{Prompt: prompt})
	if err != nil {
		t.Fatalf("encode TextGenerateInput: %v", err)
	}

	request := executioncontract.AIExecutionRequest{
		RequestID:  "gemini-live-api-test",
		Capability: CapabilityTextGenerate,
		Input:      input,
		Metadata: map[string]any{
			"test": "live-gemini-api",
		},
	}

	result, err := executor.Execute(ctx, request)
	if err != nil {
		// Public errors intentionally hide provider internals. In this opt-in
		// local integration test, print only the internal cause after redacting
		// the exact API key so authentication/model/quota/network failures can
		// be diagnosed without exposing the credential.
		var prideErr *prideerrors.Error
		if stderrors.As(err, &prideErr) && prideErr.Cause != nil {
			diagnostic := strings.ReplaceAll(prideErr.Cause.Error(), apiKey, "[REDACTED]")
			t.Logf("redacted Gemini provider cause: %s", diagnostic)
		}
		t.Fatalf("execute live Gemini request failed (model=%q; API key value omitted): %v", model, err)
	}
	if result.RequestID != request.RequestID {
		t.Fatalf("result request ID = %q, want %q", result.RequestID, request.RequestID)
	}
	if result.Capability != CapabilityTextGenerate {
		t.Fatalf("result capability = %q, want %q", result.Capability, CapabilityTextGenerate)
	}
	if result.Output == nil {
		t.Fatal("live Gemini result output is nil")
	}

	var output aiv1.TextGenerateOutput
	if err := result.Output.UnmarshalTo(&output); err != nil {
		t.Fatalf("decode TextGenerateOutput: %v", err)
	}
	if strings.TrimSpace(output.Text) == "" {
		t.Fatal("live Gemini response text is empty")
	}
	if strings.Contains(output.Text, apiKey) {
		t.Fatal("live Gemini response unexpectedly contains the API key")
	}
	if result.Metadata["provider"] != "gemini" {
		t.Fatalf("provider metadata = %v, want gemini", result.Metadata["provider"])
	}
	if result.Metadata["model"] != model {
		t.Fatalf("model metadata = %v, want %q", result.Metadata["model"], model)
	}

	t.Logf("Live Gemini API request succeeded: provider=gemini model=%s response_bytes=%d", model, len(output.Text))
}
