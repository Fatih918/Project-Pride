package gemini

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/provider"
	"pride-ai-module/selection"
)

const (
	fullE2EConfigurationRef  = "gemini.full-e2e.configuration"
	fullE2EImplementationRef = "gemini"
	fullE2ECredential        = "full-e2e-test-secret"
	fullE2EModel             = "gemini-test-model"
)

type fullE2EKeySource struct{}

func (fullE2EKeySource) GetKey(context.Context) ([]byte, error) {
	return []byte("0123456789abcdef0123456789abcdef"), nil
}

func TestGeminiFullExecutorE2EUsesSQLiteCredentialAndConfigurationStores(t *testing.T) {
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
	credentialRef, err := credentialService.Create(ctx, []byte(fullE2ECredential))
	if err != nil {
		t.Fatalf("create test credential: %v", err)
	}

	storedCredential, err := credentialDB.Get(ctx, credentialRef)
	if err != nil {
		t.Fatalf("read stored credential: %v", err)
	}
	if strings.Contains(string(storedCredential.Value), fullE2ECredential) {
		t.Fatal("credential database contains the plaintext secret")
	}

	configurationValue := configuration.AIConfiguration{
		Ref:               fullE2EConfigurationRef,
		Name:              "Gemini Full Executor E2E",
		Type:              "gemini",
		ImplementationRef: fullE2EImplementationRef,
		CredentialRef:     credentialRef,
		Enabled:           true,
		ConnectionParameters: map[string]any{
			ConnectionParameterModel: fullE2EModel,
		},
	}
	if err := configurationStore.Create(ctx, configurationValue); err != nil {
		t.Fatalf("create AI configuration: %v", err)
	}
	if err := configurationStore.Select(ctx, fullE2EConfigurationRef); err != nil {
		t.Fatalf("select AI configuration: %v", err)
	}

	selectionService := selection.NewService(selection.NewMemoryStore())
	if err := selectionService.Select(ctx, fullE2EConfigurationRef); err != nil {
		t.Fatalf("select configuration reference for executor: %v", err)
	}

	fakeClient := &fakeContentGenerator{
		response: &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: "PRIDE full E2E succeeded"}},
				},
			}},
		},
	}

	var factoryCalls int
	adapter, err := NewAdapterWithFactory(func(
		factoryContext context.Context,
		executionContext executioncontract.AIExecutionContext,
	) (provider.Provider, error) {
		if factoryContext == nil {
			t.Fatal("provider factory received nil context")
		}
		factoryCalls++
		if string(executionContext.Credential) != fullE2ECredential {
			t.Fatal("provider factory did not receive the decrypted credential")
		}
		if executionContext.Configuration.Ref != fullE2EConfigurationRef {
			t.Fatalf("configuration ref = %q, want %q",
				executionContext.Configuration.Ref, fullE2EConfigurationRef)
		}
		return &RealProvider{
			client: fakeClient,
			model:  fullE2EModel,
		}, nil
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
	if err := implementationResolver.Register(fullE2EImplementationRef, adapter); err != nil {
		t.Fatalf("register Gemini adapter: %v", err)
	}

	executor := execution.NewExecutor(
		configurationStore,
		selectionService,
		implementationResolver,
		credential.NewResolver(credentialService),
	)

	input, err := anypb.New(&aiv1.TextGenerateInput{
		Prompt: "Say hello",
	})
	if err != nil {
		t.Fatalf("encode TextGenerateInput: %v", err)
	}

	request := executioncontract.AIExecutionRequest{
		RequestID:  "gemini-full-executor-e2e",
		Capability: CapabilityTextGenerate,
		Input:      input,
		Metadata: map[string]any{
			"trace": "full-e2e-test",
		},
	}

	result, err := executor.Execute(ctx, request)
	if err != nil {
		t.Fatalf("execute full Gemini path: %v", err)
	}

	if result.RequestID != request.RequestID {
		t.Fatalf("result request ID = %q, want %q", result.RequestID, request.RequestID)
	}
	if result.Capability != CapabilityTextGenerate {
		t.Fatalf("result capability = %q, want %q", result.Capability, CapabilityTextGenerate)
	}
	if result.Output == nil {
		t.Fatal("result output is nil")
	}

	var output aiv1.TextGenerateOutput
	if err := result.Output.UnmarshalTo(&output); err != nil {
		t.Fatalf("decode TextGenerateOutput: %v", err)
	}
	if output.Text != "PRIDE full E2E succeeded" {
		t.Fatalf("output text = %q, want full E2E response", output.Text)
	}
	if result.Metadata["provider"] != "gemini" {
		t.Fatalf("provider metadata = %v, want gemini", result.Metadata["provider"])
	}
	if result.Metadata["model"] != fullE2EModel {
		t.Fatalf("model metadata = %v, want %q", result.Metadata["model"], fullE2EModel)
	}
	if fakeClient.gotModel != fullE2EModel {
		t.Fatalf("provider model = %q, want %q", fakeClient.gotModel, fullE2EModel)
	}
	if len(fakeClient.gotContents) != 1 ||
		len(fakeClient.gotContents[0].Parts) != 1 ||
		fakeClient.gotContents[0].Parts[0].Text != "Say hello" {
		t.Fatal("Gemini provider did not receive the expected prompt")
	}
	if factoryCalls != 1 {
		t.Fatalf("provider factory calls = %d, want 1", factoryCalls)
	}
}
