package execution

import (
	"bytes"
	"context"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	"pride-ai-module/implementation"
)

type credentialBoundaryImplementation struct {
	receivedRequest executioncontract.AIExecutionRequest
	receivedContext executioncontract.AIExecutionContext
	called          bool
}

func (i *credentialBoundaryImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	i.called = true

	i.receivedRequest = request
	i.receivedContext = executionContext

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     request.Input,
		Metadata: map[string]any{
			"test": "credential-boundary",
		},
	}, nil
}

func TestExecutorCredentialBoundary(t *testing.T) {
	ctx := context.Background()

	// ------------------------------------------------------------
	// 1. Create real credential through Credential Service.
	// ------------------------------------------------------------

	credentialStore := credential.NewMemoryStore()
	credentialService := credential.NewService(
		credentialStore,
	)

	expectedCredential := []byte(
		"credential-boundary-secret",
	)

	credentialRef, err := credentialService.Create(
		ctx,
		expectedCredential,
	)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if credentialRef == "" {
		t.Fatal(
			"Create() returned empty credential reference",
		)
	}

	// ------------------------------------------------------------
	// 2. Create the actual credential resolver that adapts
	//    Credential Service to the execution boundary.
	// ------------------------------------------------------------

	credentialResolver := credential.NewResolver(
		credentialService,
	)

	// ------------------------------------------------------------
	// 3. Configuration contains ONLY CredentialRef.
	//    The secret itself is not stored in configuration.
	// ------------------------------------------------------------

	config := configuration.AIConfiguration{
		Ref:               "config_credential_boundary",
		Name:              "credential-boundary-test",
		Type:              "test",
		ImplementationRef: "test.credential.boundary",
		CredentialRef:     credentialRef,
		Enabled:           true,
		ConnectionParameters: map[string]any{
			"model": "test-model",
		},
	}

	configProvider := &testConfigurationProvider{
		configuration: config,
	}

	// ------------------------------------------------------------
	// 4. Implementation captures the execution context so the
	//    test can verify that the credential reached the boundary.
	// ------------------------------------------------------------

	testImplementation := &credentialBoundaryImplementation{}

	implementationResolver := implementation.NewResolver()

	if err := implementationResolver.Register(
		config.ImplementationRef,
		testImplementation,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 5. Build the real Executor.
	// ------------------------------------------------------------

	executor := NewExecutor(
		configProvider,
		newTestSelection(
			t,
			ctx,
			config.Ref,
		),
		implementationResolver,
		credentialResolver,
	)

	// ------------------------------------------------------------
	// 6. Build normal execution input.
	// ------------------------------------------------------------

	input := &anypb.Any{
		TypeUrl: "test/input",
		Value:   []byte("test-input"),
	}

	request := executioncontract.AIExecutionRequest{
		RequestID:  "req_credential_boundary",
		Capability: "test.generate",
		Input:      input,
		Metadata: map[string]any{
			"source": "credential-boundary-test",
		},
	}

	// ------------------------------------------------------------
	// 7. Execute through the real Executor.
	// ------------------------------------------------------------

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

	// ------------------------------------------------------------
	// 8. Verify implementation was reached.
	// ------------------------------------------------------------

	if !testImplementation.called {
		t.Fatal(
			"AI implementation was not called",
		)
	}

	// ------------------------------------------------------------
	// 9. Verify Executor passed the correct credential reference
	//    to the Credential Resolver indirectly by checking that
	//    the resulting credential is correct.
	// ------------------------------------------------------------

	if !bytes.Equal(
		testImplementation.receivedContext.Credential,
		expectedCredential,
	) {
		t.Fatalf(
			"credential did not reach AIExecutionContext: got %q, want %q",
			string(testImplementation.receivedContext.Credential),
			string(expectedCredential),
		)
	}

	// ------------------------------------------------------------
	// 10. Verify the configuration reaching the implementation
	//     contains the credential reference, not the secret.
	// ------------------------------------------------------------

	if testImplementation.receivedContext.Configuration.CredentialRef != credentialRef {
		t.Fatalf(
			"unexpected credential reference: got %q, want %q",
			testImplementation.receivedContext.Configuration.CredentialRef,
			credentialRef,
		)
	}

	// ------------------------------------------------------------
	// 11. Verify the execution request itself does not contain
	//     a credential field.
	//
	//     This is enforced structurally by AIExecutionRequest,
	//     but we verify the request reaching the implementation
	//     remains unchanged.
	// ------------------------------------------------------------

	if testImplementation.receivedRequest.RequestID != request.RequestID {
		t.Fatalf(
			"request ID changed: got %q, want %q",
			testImplementation.receivedRequest.RequestID,
			request.RequestID,
		)
	}

	if testImplementation.receivedRequest.Capability != request.Capability {
		t.Fatalf(
			"capability changed: got %q, want %q",
			testImplementation.receivedRequest.Capability,
			request.Capability,
		)
	}

	if testImplementation.receivedRequest.Input != request.Input {
		t.Fatal(
			"request input changed",
		)
	}

	// ------------------------------------------------------------
	// 12. Verify credential is NOT inserted into request metadata.
	// ------------------------------------------------------------

	if value, exists := testImplementation.receivedRequest.Metadata["credential"]; exists {
		t.Fatalf(
			"credential leaked into request metadata: %v",
			value,
		)
	}

	// ------------------------------------------------------------
	// 13. Verify execution result still propagates normally.
	// ------------------------------------------------------------

	if result.RequestID != request.RequestID {
		t.Fatalf(
			"unexpected result request ID: got %q, want %q",
			result.RequestID,
			request.RequestID,
		)
	}

	if result.Capability != request.Capability {
		t.Fatalf(
			"unexpected result capability: got %q, want %q",
			result.Capability,
			request.Capability,
		)
	}

	if result.Output != request.Input {
		t.Fatal(
			"unexpected result output",
		)
	}

	if result.Metadata["test"] != "credential-boundary" {
		t.Fatalf(
			"unexpected result metadata: %v",
			result.Metadata,
		)
	}
}
