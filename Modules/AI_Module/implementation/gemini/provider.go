package gemini

import (
	"context"
	"fmt"

	"google.golang.org/genai"
	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/contracts/execution"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/provider"
)

const (
	CapabilityTextGenerate   = "text.generate"
	ConnectionParameterModel = "model"
)

type contentGenerator interface {
	GenerateContent(
		ctx context.Context,
		model string,
		contents []*genai.Content,
		config *genai.GenerateContentConfig,
	) (*genai.GenerateContentResponse, error)
}

type RealProvider struct {
	client contentGenerator
	model  string
}

func NewRealProvider(
	ctx context.Context,
	executionContext execution.AIExecutionContext,
) (*RealProvider, error) {
	if ctx == nil {
		return nil, fmt.Errorf(
			"Gemini provider context must not be nil",
		)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if len(executionContext.Credential) == 0 {
		return nil, fmt.Errorf(
			"Gemini provider credential must not be empty",
		)
	}

	model, err := resolveModel(
		executionContext.Configuration.ConnectionParameters,
	)
	if err != nil {
		return nil, err
	}

	client, err := genai.NewClient(
		ctx,
		&genai.ClientConfig{
			APIKey:  string(executionContext.Credential),
			Backend: genai.BackendGeminiAPI,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create Gemini client: %w",
			err,
		)
	}

	return &RealProvider{
		client: client.Models,
		model:  model,
	}, nil
}

func (p *RealProvider) Execute(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	if p == nil {
		return provider.Response{}, fmt.Errorf(
			"Gemini provider must not be nil",
		)
	}

	if ctx == nil {
		return provider.Response{}, fmt.Errorf(
			"Gemini provider context must not be nil",
		)
	}

	if err := ctx.Err(); err != nil {
		return provider.Response{}, err
	}

	if p.client == nil {
		return provider.Response{}, fmt.Errorf(
			"Gemini provider client must not be nil",
		)
	}

	switch request.Capability {
	case CapabilityTextGenerate:
		return p.executeTextGenerate(ctx, request)

	default:
		return provider.Response{}, fmt.Errorf(
			"Gemini capability %q is not supported",
			request.Capability,
		)
	}
}

func (p *RealProvider) executeTextGenerate(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	input, err := decodeTextGenerateInput(request.Input)
	if err != nil {
		return provider.Response{}, err
	}

	response, err := p.client.GenerateContent(
		ctx,
		p.model,
		genai.Text(input.Prompt),
		nil,
	)
	if err != nil {
		return provider.Response{}, normalizeProviderError(err)
	}

	if response == nil {
		return provider.Response{}, fmt.Errorf(
			"Gemini GenerateContent returned nil response",
		)
	}

	text := response.Text()

	if text == "" {
		return provider.Response{}, fmt.Errorf(
			"Gemini GenerateContent returned empty text",
		)
	}

	output := &aiv1.TextGenerateOutput{
		Text: text,
	}

	return provider.Response{
		Output: output,
		Metadata: map[string]any{
			"provider": "gemini",
			"model":    p.model,
		},
	}, nil
}

func decodeTextGenerateInput(
	input any,
) (*aiv1.TextGenerateInput, error) {
	value, ok := input.(*anypb.Any)
	if !ok {
		return nil, fmt.Errorf(
			"Gemini text.generate input must be protobuf Any, got %T",
			input,
		)
	}

	if value == nil {
		return nil, fmt.Errorf(
			"Gemini text.generate input Any must not be nil",
		)
	}

	result := &aiv1.TextGenerateInput{}

	if err := value.UnmarshalTo(result); err != nil {
		return nil, fmt.Errorf(
			"failed to decode Gemini text.generate input: %w",
			err,
		)
	}

	if result.Prompt == "" {
		return nil, fmt.Errorf(
			"Gemini text.generate prompt must not be empty",
		)
	}

	return result, nil
}

func resolveModel(
	parameters map[string]any,
) (string, error) {
	if parameters == nil {
		return "", fmt.Errorf(
			"Gemini connection parameters must not be nil",
		)
	}

	value, ok := parameters[ConnectionParameterModel]
	if !ok {
		return "", fmt.Errorf(
			"Gemini connection parameter %q is required",
			ConnectionParameterModel,
		)
	}

	model, ok := value.(string)
	if !ok {
		return "", fmt.Errorf(
			"Gemini connection parameter %q must be a string",
			ConnectionParameterModel,
		)
	}

	if model == "" {
		return "", fmt.Errorf(
			"Gemini connection parameter %q must not be empty",
			ConnectionParameterModel,
		)
	}

	return model, nil
}

func NewRealProviderFactory() ProviderFactory {
	return func(
		ctx context.Context,
		executionContext execution.AIExecutionContext,
	) (provider.Provider, error) {
		return NewRealProvider(
			ctx,
			executionContext,
		)
	}
}

var _ provider.Provider = (*RealProvider)(nil)
