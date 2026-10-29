package execution_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	"pride-ai-module/implementation/gemini"
	"pride-ai-module/selection"

	aiv1 "pride-ai-module/proto"
)

type geminiIntegrationConfigurationProvider struct {
	configuration configuration.AIConfiguration
}

func (p *geminiIntegrationConfigurationProvider) GetActive(
	ctx context.Context,
) (configuration.AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return configuration.AIConfiguration{}, err
	}

	return p.configuration, nil
}

func (p *geminiIntegrationConfigurationProvider) GetByRef(
	ctx context.Context,
	ref string,
) (configuration.AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return configuration.AIConfiguration{}, err
	}

	if ref != p.configuration.Ref {
		return configuration.AIConfiguration{},
			configuration.ErrConfigurationRefNotFound
	}

	return p.configuration, nil
}

var _ configuration.AIConfigurationProvider = (*geminiIntegrationConfigurationProvider)(nil)

func TestGeminiRealProviderThroughExecutor(t *testing.T) {
	if strings.TrimSpace(os.Getenv("GEMINI_RUN_EXECUTOR_REAL_API_TEST")) != "1" {
		t.Skip("set GEMINI_RUN_EXECUTOR_REAL_API_TEST=1 to run the real Gemini API integration test")
	}

	apiKey := strings.TrimSpace(
		os.Getenv("GEMINI_API_KEY"),
	)

	if apiKey == "" {
		t.Skip(
			"GEMINI_API_KEY is not set; skipping real Gemini integration test",
		)
	}

	model := strings.TrimSpace(
		os.Getenv("GEMINI_MODEL"),
	)

	if model == "" {
		t.Skip(
			"GEMINI_MODEL is not set; skipping real Gemini integration test",
		)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		60*time.Second,
	)
	defer cancel()

	// ------------------------------------------------------------
	// 1. Credential
	// ------------------------------------------------------------

	credentialStore := credential.NewMemoryStore()

	credentialService := credential.NewService(
		credentialStore,
	)

	credentialRef, err := credentialService.Create(
		ctx,
		[]byte(apiKey),
	)
	if err != nil {
		t.Fatalf(
			"failed to create credential: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 2. Configuration
	// ------------------------------------------------------------

	const (
		configurationRef  = "config_gemini_real_integration"
		implementationRef = "gemini"
	)

	aiConfiguration := configuration.AIConfiguration{
		Ref:               configurationRef,
		Name:              "Gemini Real Integration Test",
		Type:              "gemini",
		ImplementationRef: implementationRef,
		CredentialRef:     credentialRef,
		Enabled:           true,
		ConnectionParameters: map[string]any{
			gemini.ConnectionParameterModel: model,
		},
	}

	configurationProvider :=
		&geminiIntegrationConfigurationProvider{
			configuration: aiConfiguration,
		}

	// ------------------------------------------------------------
	// 3. Selection
	//
	// Selection knows ONLY ConfigurationRef.
	// ------------------------------------------------------------

	selectionStore := selection.NewMemoryStore()

	selectionService := selection.NewService(
		selectionStore,
	)

	if err := selectionService.Select(
		ctx,
		configurationRef,
	); err != nil {
		t.Fatalf(
			"failed to select Gemini configuration: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 4. Gemini Adapter
	// ------------------------------------------------------------

	geminiAdapter, err := gemini.NewAdapterWithFactory(
		gemini.NewRealProviderFactory(),
	)
	if err != nil {
		t.Fatalf(
			"failed to create Gemini adapter: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 5. Implementation Resolver
	// ------------------------------------------------------------

	implementationResolver := implementation.NewResolver()

	if err := implementationResolver.Register(
		implementationRef,
		geminiAdapter,
	); err != nil {
		t.Fatalf(
			"failed to register Gemini implementation: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 6. Credential Resolver
	// ------------------------------------------------------------

	credentialResolver :=
		execution.NewCredentialServiceResolver(
			credentialService,
		)

	// ------------------------------------------------------------
	// 7. Executor
	// ------------------------------------------------------------

	executor := execution.NewExecutor(
		configurationProvider,
		selectionService,
		implementationResolver,
		credentialResolver,
	)

	// ------------------------------------------------------------
	// 8. Canonical text.generate input
	// ------------------------------------------------------------

	input := &aiv1.TextGenerateInput{
		Prompt: "Reply with exactly: PRIDE_EXECUTOR_REAL_API_OK",
	}

	inputAny, err := anypb.New(input)
	if err != nil {
		t.Fatalf(
			"failed to create TextGenerateInput Any: %v",
			err,
		)
	}

	request := executioncontract.AIExecutionRequest{
		RequestID:  "gemini-executor-real-api-test",
		Capability: gemini.CapabilityTextGenerate,
		Input:      inputAny,
		Metadata: map[string]any{
			"test": "13.6.6",
		},
	}

	// ------------------------------------------------------------
	// 9. Execute
	//
	// This is the complete real path:
	//
	// Selection
	//   -> Configuration
	//   -> Implementation Resolver
	//   -> Credential Resolver
	//   -> Gemini Adapter
	//   -> RealProvider
	//   -> Gemini API
	// ------------------------------------------------------------

	result, err := executor.Execute(
		ctx,
		request,
	)
	if err != nil {
		t.Fatalf(
			"real Gemini execution through Executor failed: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 10. Validate result
	// ------------------------------------------------------------

	if result.RequestID != request.RequestID {
		t.Fatalf(
			"result RequestID = %q, want %q",
			result.RequestID,
			request.RequestID,
		)
	}

	if result.Capability != request.Capability {
		t.Fatalf(
			"result Capability = %q, want %q",
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
		"Gemini model: %s",
		model,
	)

	t.Logf(
		"Gemini output: %q",
		output.Text,
	)

	// ------------------------------------------------------------
	// 11. Validate normalized metadata
	// ------------------------------------------------------------

	if result.Metadata["provider"] != "gemini" {
		t.Fatalf(
			"result provider metadata = %v, want gemini",
			result.Metadata["provider"],
		)
	}

	if result.Metadata["model"] != model {
		t.Fatalf(
			"result model metadata = %v, want %q",
			result.Metadata["model"],
			model,
		)
	}
}
