package execution

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	selectioncontract "pride-ai-module/contracts/selection"
	"pride-ai-module/credential"
	"pride-ai-module/implementation"
	"pride-ai-module/selection"
)

type testConfigurationProvider struct {
	configuration configuration.AIConfiguration
	err           error
}

func (p *testConfigurationProvider) GetActive(
	ctx context.Context,
) (configuration.AIConfiguration, error) {
	if p.err != nil {
		return configuration.AIConfiguration{}, p.err
	}

	return p.configuration, nil
}

func (p *testConfigurationProvider) GetByRef(
	ctx context.Context,
	ref string,
) (configuration.AIConfiguration, error) {
	if p.err != nil {
		return configuration.AIConfiguration{}, p.err
	}

	if ref != p.configuration.Ref {
		return configuration.AIConfiguration{}, errors.New(
			"configuration not found",
		)
	}

	return p.configuration, nil
}

type testCredentialResolver struct {
	value []byte
	err   error

	receivedRef string
}

func (r *testCredentialResolver) Resolve(
	ctx context.Context,
	ref string,
) ([]byte, error) {
	r.receivedRef = ref

	if r.err != nil {
		return nil, r.err
	}

	return r.value, nil
}

type testImplementation struct {
	result executioncontract.AIExecutionResult
	err    error

	receivedRequest executioncontract.AIExecutionRequest
	receivedContext executioncontract.AIExecutionContext
	receivedGoCtx   context.Context
}

func (i *testImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	i.receivedGoCtx = ctx
	i.receivedRequest = request
	i.receivedContext = executionContext

	if i.err != nil {
		return executioncontract.AIExecutionResult{}, i.err
	}

	return i.result, nil
}

func newTestCredentialResolver() *testCredentialResolver {
	return &testCredentialResolver{
		value: []byte("test-api-key"),
	}
}

func newTestSelection(
	t *testing.T,
	ctx context.Context,
	configurationRef string,
) selectioncontract.ConfigurationSelector {
	t.Helper()

	store := selection.NewMemoryStore()
	service := selection.NewService(store)

	if err := service.Select(ctx, configurationRef); err != nil {
		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}

	return service
}

func TestExecutorExecute(t *testing.T) {
	ctx := context.Background()

	input := &anypb.Any{
		TypeUrl: "test/input",
		Value:   []byte("input"),
	}

	output := &anypb.Any{
		TypeUrl: "test/output",
		Value:   []byte("output"),
	}

	request := executioncontract.AIExecutionRequest{
		RequestID:  "req_001",
		Capability: "text.generate",
		Input:      input,
		Metadata: map[string]any{
			"source": "test",
		},
	}

	expectedResult := executioncontract.AIExecutionResult{
		RequestID:  "req_001",
		Capability: "text.generate",
		Output:     output,
		Metadata: map[string]any{
			"provider": "test",
		},
	}

	config := configuration.AIConfiguration{
		Ref:               "config_test",
		Name:              "test-config",
		Type:              "test",
		ImplementationRef: "test.implementation",
		ConnectionParameters: map[string]any{
			"model": "test-model",
		},
		CredentialRef: "cred_test_001",
		Enabled:       true,
	}

	impl := &testImplementation{
		result: expectedResult,
	}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	credentialResolver := newTestCredentialResolver()

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		config.ImplementationRef,
		impl,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	selector := newTestSelection(
		t,
		ctx,
		config.Ref,
	)

	executor := NewExecutor(
		provider,
		selector,
		resolver,
		credentialResolver,
	)

	result, err := executor.Execute(
		ctx,
		request,
	)
	if err != nil {
		t.Fatalf(
			"Execute() failed: %v",
			err,
		)
	}

	if result.RequestID != expectedResult.RequestID {
		t.Fatalf(
			"unexpected RequestID: got %q, want %q",
			result.RequestID,
			expectedResult.RequestID,
		)
	}

	if result.Capability != expectedResult.Capability {
		t.Fatalf(
			"unexpected Capability: got %q, want %q",
			result.Capability,
			expectedResult.Capability,
		)
	}

	if result.Output != expectedResult.Output {
		t.Fatal("executor returned unexpected output")
	}

	if impl.receivedGoCtx != ctx {
		t.Fatal("executor did not forward context")
	}

	if impl.receivedRequest.RequestID != request.RequestID {
		t.Fatalf(
			"unexpected implementation RequestID: got %q, want %q",
			impl.receivedRequest.RequestID,
			request.RequestID,
		)
	}

	if impl.receivedRequest.Capability != request.Capability {
		t.Fatalf(
			"unexpected implementation Capability: got %q, want %q",
			impl.receivedRequest.Capability,
			request.Capability,
		)
	}

	if impl.receivedContext.Configuration.Ref != config.Ref {
		t.Fatalf(
			"unexpected configuration reference: got %q, want %q",
			impl.receivedContext.Configuration.Ref,
			config.Ref,
		)
	}

	if impl.receivedContext.Configuration.ImplementationRef !=
		config.ImplementationRef {
		t.Fatalf(
			"unexpected implementation configuration: got %q, want %q",
			impl.receivedContext.Configuration.ImplementationRef,
			config.ImplementationRef,
		)
	}

	if string(impl.receivedContext.Credential) !=
		string(credentialResolver.value) {
		t.Fatalf(
			"unexpected implementation credential: got %q, want %q",
			string(impl.receivedContext.Credential),
			string(credentialResolver.value),
		)
	}

	if credentialResolver.receivedRef != config.CredentialRef {
		t.Fatalf(
			"unexpected credential reference: got %q, want %q",
			credentialResolver.receivedRef,
			config.CredentialRef,
		)
	}
}

func TestExecutorRejectsDisabledConfiguration(t *testing.T) {
	ctx := context.Background()

	config := configuration.AIConfiguration{
		Ref:               "config_disabled",
		Name:              "disabled-config",
		Type:              "test",
		ImplementationRef: "test.implementation",
		CredentialRef:     "cred_test",
		Enabled:           false,
	}

	impl := &testImplementation{}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		config.ImplementationRef,
		impl,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	credentialResolver := newTestCredentialResolver()

	selector := newTestSelection(
		t,
		ctx,
		config.Ref,
	)

	executor := NewExecutor(
		provider,
		selector,
		resolver,
		credentialResolver,
	)

	_, err := executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{
			RequestID:  "req_disabled",
			Capability: "test",
		},
	)

	if err == nil {
		t.Fatal(
			"expected disabled configuration to return an error",
		)
	}
}

func TestExecutorPropagatesSelectionError(t *testing.T) {
	ctx := context.Background()

	config := configuration.AIConfiguration{
		Ref:               "config_test",
		Name:              "test-config",
		Type:              "test",
		ImplementationRef: "test.implementation",
		CredentialRef:     "cred_test",
		Enabled:           true,
	}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	resolver := implementation.NewResolver()

	credentialResolver := newTestCredentialResolver()

	store := selection.NewMemoryStore()
	selector := selection.NewService(store)

	executor := NewExecutor(
		provider,
		selector,
		resolver,
		credentialResolver,
	)

	_, err := executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{},
	)

	if err == nil {
		t.Fatal("expected selection error")
	}

	if !errors.Is(err, selection.ErrConfigurationNotSelected) {
		t.Fatalf(
			"unexpected error: got %v",
			err,
		)
	}
}

func TestExecutorPropagatesConfigurationError(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New(
		"configuration unavailable",
	)

	provider := &testConfigurationProvider{
		err: expectedErr,
	}

	resolver := implementation.NewResolver()

	credentialResolver := newTestCredentialResolver()

	selector := newTestSelection(
		t,
		ctx,
		"config_test",
	)

	executor := NewExecutor(
		provider,
		selector,
		resolver,
		credentialResolver,
	)

	_, err := executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{},
	)

	if err == nil {
		t.Fatal("expected configuration error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"unexpected error: got %v",
			err,
		)
	}
}

func TestExecutorPropagatesResolutionError(t *testing.T) {
	ctx := context.Background()

	config := configuration.AIConfiguration{
		Ref:               "config_test",
		Name:              "test-config",
		Type:              "test",
		ImplementationRef: "missing.implementation",
		CredentialRef:     "cred_test",
		Enabled:           true,
	}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	resolver := implementation.NewResolver()

	credentialResolver := newTestCredentialResolver()

	selector := newTestSelection(
		t,
		ctx,
		config.Ref,
	)

	executor := NewExecutor(
		provider,
		selector,
		resolver,
		credentialResolver,
	)

	_, err := executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{},
	)

	if err == nil {
		t.Fatal("expected implementation resolution error")
	}
}

func TestExecutorPropagatesCredentialResolutionError(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New(
		"credential unavailable",
	)

	config := configuration.AIConfiguration{
		Ref:               "config_test",
		Name:              "test-config",
		Type:              "test",
		ImplementationRef: "test.implementation",
		CredentialRef:     "cred_test",
		Enabled:           true,
	}

	impl := &testImplementation{}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		config.ImplementationRef,
		impl,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	credentialResolver := &testCredentialResolver{
		err: expectedErr,
	}

	selector := newTestSelection(
		t,
		ctx,
		config.Ref,
	)

	executor := NewExecutor(
		provider,
		selector,
		resolver,
		credentialResolver,
	)

	_, err := executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{},
	)

	if err == nil {
		t.Fatal("expected credential resolution error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"unexpected error: got %v",
			err,
		)
	}
}

func TestExecutorPropagatesImplementationError(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New(
		"implementation execution failed",
	)

	config := configuration.AIConfiguration{
		Ref:               "config_test",
		Name:              "test-config",
		Type:              "test",
		ImplementationRef: "test.implementation",
		CredentialRef:     "cred_test",
		Enabled:           true,
	}

	impl := &testImplementation{
		err: expectedErr,
	}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		config.ImplementationRef,
		impl,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	credentialResolver := newTestCredentialResolver()

	selector := newTestSelection(
		t,
		ctx,
		config.Ref,
	)

	executor := NewExecutor(
		provider,
		selector,
		resolver,
		credentialResolver,
	)

	_, err := executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{},
	)

	if err == nil {
		t.Fatal("expected implementation execution error")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"unexpected error: got %v",
			err,
		)
	}
}

func TestExecutorRejectsNilCredentialResolver(t *testing.T) {
	ctx := context.Background()

	config := configuration.AIConfiguration{
		Ref:               "config_test",
		Name:              "test-config",
		Type:              "test",
		ImplementationRef: "test.implementation",
		CredentialRef:     "cred_test",
		Enabled:           true,
	}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		config.ImplementationRef,
		&testImplementation{},
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	selector := newTestSelection(
		t,
		ctx,
		config.Ref,
	)

	executor := NewExecutor(
		provider,
		selector,
		resolver,
		nil,
	)

	_, err := executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{},
	)

	if err == nil {
		t.Fatal("expected nil credential resolver error")
	}
}

func TestExecutorRejectsNilSelection(t *testing.T) {
	ctx := context.Background()

	config := configuration.AIConfiguration{
		Ref:               "config_test",
		Name:              "test-config",
		Type:              "test",
		ImplementationRef: "test.implementation",
		CredentialRef:     "cred_test",
		Enabled:           true,
	}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		config.ImplementationRef,
		&testImplementation{},
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	credentialResolver := newTestCredentialResolver()

	executor := NewExecutor(
		provider,
		nil,
		resolver,
		credentialResolver,
	)

	_, err := executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{},
	)

	if err == nil {
		t.Fatal("expected nil selection error")
	}
}

func TestServiceCredentialResolver(t *testing.T) {
	ctx := context.Background()

	store := credential.NewMemoryStore()
	service := credential.NewService(store)

	secret := []byte("test-api-key")

	ref, err := service.Create(
		ctx,
		secret,
	)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	resolver := NewCredentialServiceResolver(service)

	result, err := resolver.Resolve(
		ctx,
		ref,
	)
	if err != nil {
		t.Fatalf(
			"Resolve() failed: %v",
			err,
		)
	}

	if string(result) != string(secret) {
		t.Fatalf(
			"unexpected credential value: got %q, want %q",
			string(result),
			string(secret),
		)
	}
}
func TestExecutorRejectsMissingCredential(t *testing.T) {
	ctx := context.Background()

	config := configuration.AIConfiguration{
		Ref:               "config_missing_credential",
		Name:              "missing-credential-config",
		Type:              "test",
		ImplementationRef: "test.implementation",
		CredentialRef:     "cred_missing",
		Enabled:           true,
	}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	impl := &testImplementation{}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		config.ImplementationRef,
		impl,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	credentialStore := credential.NewMemoryStore()
	credentialService := credential.NewService(
		credentialStore,
	)

	credentialResolver := NewCredentialServiceResolver(
		credentialService,
	)

	selector := newTestSelection(
		t,
		ctx,
		config.Ref,
	)

	executor := NewExecutor(
		provider,
		selector,
		resolver,
		credentialResolver,
	)

	_, err := executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{
			RequestID:  "req_missing_credential",
			Capability: "text.generate",
		},
	)

	if err == nil {
		t.Fatal(
			"expected missing credential to return an error",
		)
	}

	if impl.receivedRequest.RequestID != "" {
		t.Fatal(
			"implementation must not be executed when credential is missing",
		)
	}
}

func TestExecutorWithoutCredentialWithResolver(t *testing.T) {
	ctx := context.Background()

	config := configuration.AIConfiguration{
		Ref:               "config_no_credential",
		Name:              "no-credential-config",
		Type:              "local",
		ImplementationRef: "test.implementation",
		CredentialRef:     "",
		Enabled:           true,
	}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	impl := &credentialCapturingImplementation{}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		config.ImplementationRef,
		impl,
	); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	credentialResolver := &testCredentialResolver{}

	executor := NewExecutor(
		provider,
		newTestSelection(
			t,
			ctx,
			config.Ref,
		),
		resolver,
		credentialResolver,
	)

	request := executioncontract.AIExecutionRequest{
		RequestID:  "req_no_credential",
		Capability: "text.generate",
	}

	_, err := executor.Execute(
		ctx,
		request,
	)
	if err != nil {
		t.Fatalf(
			"Execute() returned unexpected error: %v",
			err,
		)
	}

	if !impl.called {
		t.Fatal(
			"implementation must be executed without credential",
		)
	}

	if impl.receivedContext.Credential != nil {
		t.Fatalf(
			"expected nil credential, got %q",
			string(impl.receivedContext.Credential),
		)
	}
}

func TestExecutorWithoutCredentialWithoutResolver(t *testing.T) {
	ctx := context.Background()

	config := configuration.AIConfiguration{
		Ref:               "config_no_credential_no_resolver",
		Name:              "no-credential-no-resolver-config",
		Type:              "local",
		ImplementationRef: "test.implementation",
		CredentialRef:     "",
		Enabled:           true,
	}

	provider := &testConfigurationProvider{
		configuration: config,
	}

	impl := &credentialCapturingImplementation{}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		config.ImplementationRef,
		impl,
	); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	executor := NewExecutor(
		provider,
		newTestSelection(
			t,
			ctx,
			config.Ref,
		),
		resolver,
		nil,
	)

	request := executioncontract.AIExecutionRequest{
		RequestID:  "req_no_credential_no_resolver",
		Capability: "text.generate",
	}

	_, err := executor.Execute(
		ctx,
		request,
	)
	if err != nil {
		t.Fatalf(
			"Execute() returned unexpected error: %v",
			err,
		)
	}

	if !impl.called {
		t.Fatal(
			"implementation must be executed without credential resolver",
		)
	}

	if impl.receivedContext.Credential != nil {
		t.Fatalf(
			"expected nil credential, got %q",
			string(impl.receivedContext.Credential),
		)
	}
}

type credentialCapturingImplementation struct {
	called          bool
	receivedContext executioncontract.AIExecutionContext
}

func (i *credentialCapturingImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	i.called = true
	i.receivedContext = executionContext

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     request.Input,
		Metadata:   request.Metadata,
	}, nil
}

var _ implementation.AIImplementation =
	(*credentialCapturingImplementation)(nil)
	
var _ executioncontract.CredentialResolver = (*testCredentialResolver)(nil)
