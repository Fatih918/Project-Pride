package gemini

import (
	"context"
	"testing"

	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	"pride-ai-module/contracts/execution"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/provider"
)

// This test covers the successful adapter -> cached provider -> Gemini
// provider execution path without making a live network request.
func TestAdapterFactoryExecutesTextGenerateEndToEnd(t *testing.T) {
	fakeClient := &fakeContentGenerator{
		response: &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: "Hello from the Gemini adapter"}},
				},
			}},
		},
	}

	var factoryCalls int
	adapter, err := NewAdapterWithFactory(func(
		ctx context.Context,
		executionContext execution.AIExecutionContext,
	) (provider.Provider, error) {
		factoryCalls++
		if string(executionContext.Credential) != "test-credential" {
			t.Fatal("provider factory did not receive the resolved credential")
		}
		return &RealProvider{
			client: fakeClient,
			model:  "test-model",
		}, nil
	})
	if err != nil {
		t.Fatalf("NewAdapterWithFactory returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Errorf("adapter Close returned error: %v", err)
		}
	})

	input, err := anypb.New(&aiv1.TextGenerateInput{Prompt: "Say hello"})
	if err != nil {
		t.Fatalf("failed to pack input: %v", err)
	}

	result, err := adapter.Execute(
		context.Background(),
		execution.AIExecutionRequest{
			RequestID:  "request-gemini-e2e",
			Capability: CapabilityTextGenerate,
			Input:      input,
			Metadata:   map[string]any{"trace": "test"},
		},
		execution.AIExecutionContext{
			Configuration: configuration.AIConfiguration{
				Ref:               "config-gemini-e2e",
				ImplementationRef: "gemini",
				ConnectionParameters: map[string]any{
					ConnectionParameterModel: "test-model",
				},
				CredentialRef: "credential-gemini-test",
				Enabled:       true,
			},
			Credential: []byte("test-credential"),
		},
	)
	if err != nil {
		t.Fatalf("Adapter.Execute returned error: %v", err)
	}

	if result.RequestID != "request-gemini-e2e" {
		t.Fatalf("RequestID = %q, want %q", result.RequestID, "request-gemini-e2e")
	}
	if result.Capability != CapabilityTextGenerate {
		t.Fatalf("Capability = %q, want %q", result.Capability, CapabilityTextGenerate)
	}
	if result.Output == nil {
		t.Fatal("expected non-nil result output")
	}

	var output aiv1.TextGenerateOutput
	if err := result.Output.UnmarshalTo(&output); err != nil {
		t.Fatalf("failed to unpack output: %v", err)
	}
	if output.Text != "Hello from the Gemini adapter" {
		t.Fatalf("output text = %q, want successful generated text", output.Text)
	}
	if fakeClient.gotModel != "test-model" {
		t.Fatalf("provider model = %q, want %q", fakeClient.gotModel, "test-model")
	}
	if len(fakeClient.gotContents) != 1 ||
		len(fakeClient.gotContents[0].Parts) != 1 ||
		fakeClient.gotContents[0].Parts[0].Text != "Say hello" {
		t.Fatal("provider did not receive the expected prompt")
	}
	if factoryCalls != 1 {
		t.Fatalf("provider factory calls = %d, want 1", factoryCalls)
	}
}
