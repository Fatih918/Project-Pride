package gemini

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	"pride-ai-module/contracts/execution"
	aiv1 "pride-ai-module/proto"
)

func TestRealGeminiAPI(t *testing.T) {
	apiKey := strings.TrimSpace(
		os.Getenv("GEMINI_API_KEY"),
	)

	if apiKey == "" {
		t.Skip(
			"GEMINI_API_KEY is not set; skipping real Gemini API test",
		)
	}

	model := strings.TrimSpace(
		os.Getenv("GEMINI_MODEL"),
	)

	if model == "" {
		t.Skip(
			"GEMINI_MODEL is not set; skipping real Gemini API test",
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		60*time.Second,
	)
	defer cancel()

	input := &aiv1.TextGenerateInput{
		Prompt: "Reply with exactly: PRIDE_REAL_API_OK",
	}

	inputAny, err := anypb.New(input)
	if err != nil {
		t.Fatalf(
			"failed to create input Any: %v",
			err,
		)
	}

	executionContext := execution.AIExecutionContext{
		Configuration: configuration.AIConfiguration{
			Ref:               "config-gemini-real-test",
			Name:              "Gemini Real API Test",
			Type:              "ai",
			ImplementationRef: "gemini",
			ConnectionParameters: map[string]any{
				ConnectionParameterModel: model,
			},
			CredentialRef: "credential-gemini-real-test",
			Enabled:       true,
		},
		Credential: []byte(apiKey),
	}

	request := execution.AIExecutionRequest{
		RequestID:  "real-gemini-api-test",
		Capability: CapabilityTextGenerate,
		Input:      inputAny,
	}

	adapter, err := NewAdapterWithFactory(
		NewRealProviderFactory(),
	)
	if err != nil {
		t.Fatalf(
			"NewAdapterWithFactory returned error: %v",
			err,
		)
	}

	result, err := adapter.Execute(
		ctx,
		request,
		executionContext,
	)
	if err != nil {
		t.Fatalf(
			"real Gemini execution failed: %v",
			err,
		)
	}

	if result.RequestID != request.RequestID {
		t.Fatalf(
			"RequestID = %q, want %q",
			result.RequestID,
			request.RequestID,
		)
	}

	if result.Capability != request.Capability {
		t.Fatalf(
			"Capability = %q, want %q",
			result.Capability,
			request.Capability,
		)
	}

	if result.Output == nil {
		t.Fatal("result Output is nil")
	}

	output := &aiv1.TextGenerateOutput{}

	if err := result.Output.UnmarshalTo(output); err != nil {
		t.Fatalf(
			"failed to decode TextGenerateOutput: %v",
			err,
		)
	}

	if strings.TrimSpace(output.Text) == "" {
		t.Fatal("Gemini returned empty text")
	}

	t.Logf(
		"Gemini real API returned text: %q",
		output.Text,
	)

	if result.Metadata["provider"] != "gemini" {
		t.Fatalf(
			"metadata provider = %v, want gemini",
			result.Metadata["provider"],
		)
	}

	if result.Metadata["model"] != model {
		t.Fatalf(
			"metadata model = %v, want %q",
			result.Metadata["model"],
			model,
		)
	}
}
