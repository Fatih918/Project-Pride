package execution_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	prideerrors "pride-ai-module/errors"
	execution "pride-ai-module/execution"
	"pride-ai-module/implementation"
	"pride-ai-module/implementation/gemini"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/selection"
)

func TestGeminiInvalidCredentialThroughExecutor(t *testing.T) {
	ctx := context.Background()

	invalidCredential := []byte(
		"invalid-pride-gemini-credential",
	)

	credentialStore := credential.NewMemoryStore()
	credentialService := credential.NewService(
		credentialStore,
	)

	credentialRef, err := credentialService.Create(
		ctx,
		invalidCredential,
	)
	if err != nil {
		t.Fatalf(
			"Create() credential failed: %v",
			err,
		)
	}

	const configurationRef = "config_gemini_invalid_auth"
	const implementationRef = "gemini"

	model := "gemini-2.5-flash"

	config := configuration.AIConfiguration{
		Ref:               configurationRef,
		Name:              "gemini-invalid-auth-test",
		Type:              "gemini",
		ImplementationRef: implementationRef,
		CredentialRef:     credentialRef,
		Enabled:           true,
		ConnectionParameters: map[string]any{
			"model": model,
		},
	}

	configProvider := &geminiAuthenticationConfigurationProvider{
		configuration: config,
	}

	selectorStore := selection.NewMemoryStore()
	selectorService := selection.NewService(
		selectorStore,
	)

	if err := selectorService.Select(
		ctx,
		configurationRef,
	); err != nil {
		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}

	adapter, err := gemini.NewAdapterWithFactory(
		gemini.NewRealProviderFactory(),
	)
	if err != nil {
		t.Fatalf(
			"NewAdapterWithFactory() failed: %v",
			err,
		)
	}

	implementationResolver := implementation.NewResolver()

	if err := implementationResolver.Register(
		implementationRef,
		adapter,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	credentialResolver :=
		execution.NewCredentialServiceResolver(
			credentialService,
		)

	executor := execution.NewExecutor(
		configProvider,
		selectorService,
		implementationResolver,
		credentialResolver,
	)

	input := &aiv1.TextGenerateInput{
		Prompt: "Reply with exactly: AUTH_TEST",
	}

	inputAny, err := anypb.New(input)
	if err != nil {
		t.Fatalf(
			"anypb.New() failed: %v",
			err,
		)
	}

	request := executioncontract.AIExecutionRequest{
		RequestID:  "req_gemini_invalid_auth",
		Capability: gemini.CapabilityTextGenerate,
		Input:      inputAny,
	}

	_, err = executor.Execute(
		ctx,
		request,
	)

	if err == nil {
		t.Fatal(
			"expected invalid Gemini credential to return an error",
		)
	}

	var prideErr *prideerrors.Error

	if !errors.As(err, &prideErr) {
		t.Fatalf(
			"expected PRIDE error, got %T: %v",
			err,
			err,
		)
	}

	if prideErr.Code != prideerrors.CodeAuthenticationError {
		t.Fatalf(
			"unexpected error code: got %q, want %q",
			prideErr.Code,
			prideerrors.CodeAuthenticationError,
		)
	}

	if prideErr.Message !=
		"Gemini provider authentication failed" {
		t.Fatalf(
			"unexpected error message: got %q",
			prideErr.Message,
		)
	}

	if prideErr.RequestID != "" {
		t.Fatalf(
			"request ID should not be assigned by provider layer: got %q",
			prideErr.RequestID,
		)
	}

	if prideErr.Cause == nil {
		t.Fatal(
			"expected underlying Gemini API error as cause",
		)
	}
}

type geminiAuthenticationConfigurationProvider struct {
	configuration configuration.AIConfiguration
}

func (p *geminiAuthenticationConfigurationProvider) GetActive(
	ctx context.Context,
) (configuration.AIConfiguration, error) {
	return p.configuration, nil
}

func (p *geminiAuthenticationConfigurationProvider) GetByRef(
	ctx context.Context,
	ref string,
) (configuration.AIConfiguration, error) {
	if ref != p.configuration.Ref {
		return configuration.AIConfiguration{},
			errors.New("configuration not found")
	}

	return p.configuration, nil
}

var _ configuration.AIConfigurationProvider = (*geminiAuthenticationConfigurationProvider)(nil)
