package gemini

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	"pride-ai-module/contracts/execution"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/provider"
)

type fakeContentGenerator struct {
	response *genai.GenerateContentResponse
	err      error

	gotModel    string
	gotContents []*genai.Content
}

func (f *fakeContentGenerator) GenerateContent(
	ctx context.Context,
	model string,
	contents []*genai.Content,
	config *genai.GenerateContentConfig,
) (*genai.GenerateContentResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	f.gotModel = model
	f.gotContents = contents

	return f.response, f.err
}

func newTextGenerateRequest(
	t *testing.T,
	prompt string,
) provider.Request {
	t.Helper()

	input := &aiv1.TextGenerateInput{
		Prompt: prompt,
	}

	inputAny, err := anypb.New(input)
	if err != nil {
		t.Fatalf(
			"failed to create input Any: %v",
			err,
		)
	}

	return provider.Request{
		Capability: CapabilityTextGenerate,
		Input:      inputAny,
	}
}

func TestRealProviderImplementsProvider(t *testing.T) {
	var _ provider.Provider = (*RealProvider)(nil)
}

func TestRealProviderExecuteTextGenerate(t *testing.T) {
	fake := &fakeContentGenerator{
		response: &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{
								Text: "Hello from Gemini",
							},
						},
					},
				},
			},
		},
	}

	p := &RealProvider{
		client: fake,
		model:  "test-model",
	}

	request := newTextGenerateRequest(
		t,
		"Say hello",
	)

	response, err := p.Execute(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf(
			"Execute returned unexpected error: %v",
			err,
		)
	}

	if response.Output == nil {
		t.Fatal("expected output, got nil")
	}

	output, ok := response.Output.(*aiv1.TextGenerateOutput)
	if !ok {
		t.Fatalf(
			"expected *TextGenerateOutput, got %T",
			response.Output,
		)
	}

	if output.Text != "Hello from Gemini" {
		t.Fatalf(
			"expected generated text %q, got %q",
			"Hello from Gemini",
			output.Text,
		)
	}

	if fake.gotModel != "test-model" {
		t.Fatalf(
			"expected model %q, got %q",
			"test-model",
			fake.gotModel,
		)
	}

	if len(fake.gotContents) != 1 {
		t.Fatalf(
			"expected 1 content, got %d",
			len(fake.gotContents),
		)
	}

	if len(fake.gotContents[0].Parts) != 1 {
		t.Fatalf(
			"expected 1 part, got %d",
			len(fake.gotContents[0].Parts),
		)
	}

	if fake.gotContents[0].Parts[0].Text != "Say hello" {
		t.Fatalf(
			"expected prompt %q, got %q",
			"Say hello",
			fake.gotContents[0].Parts[0].Text,
		)
	}

	if response.Metadata["provider"] != "gemini" {
		t.Fatalf(
			"expected provider metadata %q, got %v",
			"gemini",
			response.Metadata["provider"],
		)
	}

	if response.Metadata["model"] != "test-model" {
		t.Fatalf(
			"expected model metadata %q, got %v",
			"test-model",
			response.Metadata["model"],
		)
	}
}

func TestRealProviderExecuteUnsupportedCapability(t *testing.T) {
	p := &RealProvider{
		client: &fakeContentGenerator{},
		model:  "test-model",
	}

	_, err := p.Execute(
		context.Background(),
		provider.Request{
			Capability: "embedding.generate",
		},
	)

	if err == nil {
		t.Fatal("expected unsupported capability error")
	}
}

func TestRealProviderExecuteRejectsNilProvider(t *testing.T) {
	var p *RealProvider

	_, err := p.Execute(
		context.Background(),
		provider.Request{
			Capability: CapabilityTextGenerate,
		},
	)

	if err == nil {
		t.Fatal("expected nil provider error")
	}
}

func TestRealProviderExecuteRejectsNilContext(t *testing.T) {
	p := &RealProvider{
		client: &fakeContentGenerator{},
		model:  "test-model",
	}

	_, err := p.Execute(
		context.TODO(),
		provider.Request{
			Capability: CapabilityTextGenerate,
		},
	)

	if err == nil {
		t.Fatal("expected nil context error")
	}
}

func TestRealProviderExecuteRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &RealProvider{
		client: &fakeContentGenerator{
			response: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{
								{Text: "should not execute"},
							},
						},
					},
				},
			},
		},
		model: "test-model",
	}

	_, err := p.Execute(
		ctx,
		newTextGenerateRequest(t, "hello"),
	)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"expected context.Canceled, got %v",
			err,
		)
	}
}

func TestRealProviderExecutePropagatesGeminiError(t *testing.T) {
	expectedErr := errors.New("Gemini API failure")

	p := &RealProvider{
		client: &fakeContentGenerator{
			err: expectedErr,
		},
		model: "test-model",
	}

	_, err := p.Execute(
		context.Background(),
		newTextGenerateRequest(t, "hello"),
	)

	if err == nil {
		t.Fatal("expected Gemini error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected wrapped Gemini error, got %v",
			err,
		)
	}
}

func TestRealProviderExecuteRejectsNilClient(t *testing.T) {
	p := &RealProvider{
		model: "test-model",
	}

	_, err := p.Execute(
		context.Background(),
		newTextGenerateRequest(t, "hello"),
	)

	if err == nil {
		t.Fatal("expected nil client error")
	}
}

func TestRealProviderExecuteRejectsInvalidInput(t *testing.T) {
	p := &RealProvider{
		client: &fakeContentGenerator{},
		model:  "test-model",
	}

	_, err := p.Execute(
		context.Background(),
		provider.Request{
			Capability: CapabilityTextGenerate,
			Input:      "not protobuf",
		},
	)

	if err == nil {
		t.Fatal("expected invalid input error")
	}
}

func TestRealProviderExecuteRejectsEmptyPrompt(t *testing.T) {
	request := newTextGenerateRequest(t, "")

	p := &RealProvider{
		client: &fakeContentGenerator{},
		model:  "test-model",
	}

	_, err := p.Execute(
		context.Background(),
		request,
	)

	if err == nil {
		t.Fatal("expected empty prompt error")
	}
}

func TestRealProviderExecuteRejectsNilResponse(t *testing.T) {
	p := &RealProvider{
		client: &fakeContentGenerator{
			response: nil,
		},
		model: "test-model",
	}

	_, err := p.Execute(
		context.Background(),
		newTextGenerateRequest(t, "hello"),
	)

	if err == nil {
		t.Fatal("expected nil response error")
	}
}

func TestRealProviderExecuteRejectsEmptyResponseText(t *testing.T) {
	p := &RealProvider{
		client: &fakeContentGenerator{
			response: &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{
						Content: &genai.Content{
							Parts: []*genai.Part{},
						},
					},
				},
			},
		},
		model: "test-model",
	}

	_, err := p.Execute(
		context.Background(),
		newTextGenerateRequest(t, "hello"),
	)

	if err == nil {
		t.Fatal("expected empty response text error")
	}
}

func TestResolveModel(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]any
		want       string
		wantErr    bool
	}{
		{
			name: "valid",
			parameters: map[string]any{
				ConnectionParameterModel: "gemini-test-model",
			},
			want: "gemini-test-model",
		},
		{
			name:       "nil parameters",
			parameters: nil,
			wantErr:    true,
		},
		{
			name:       "missing model",
			parameters: map[string]any{},
			wantErr:    true,
		},
		{
			name: "wrong type",
			parameters: map[string]any{
				ConnectionParameterModel: 123,
			},
			wantErr: true,
		},
		{
			name: "empty model",
			parameters: map[string]any{
				ConnectionParameterModel: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveModel(tt.parameters)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"expected model %q, got %q",
					tt.want,
					got,
				)
			}
		})
	}
}

func TestNewRealProviderRejectsEmptyCredential(t *testing.T) {
	_, err := NewRealProvider(
		context.Background(),
		execution.AIExecutionContext{},
	)

	if err == nil {
		t.Fatal("expected empty credential error")
	}
}

func TestNewRealProviderRejectsMissingModel(t *testing.T) {
	_, err := NewRealProvider(
		context.Background(),
		execution.AIExecutionContext{
			Credential: []byte("test-key"),
			Configuration: configuration.AIConfiguration{
				ConnectionParameters: map[string]any{},
			},
		},
	)

	if err == nil {
		t.Fatal("expected missing model error")
	}
}
