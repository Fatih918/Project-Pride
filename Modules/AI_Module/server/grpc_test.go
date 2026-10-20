package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/selection"
	grpcadapter "pride-ai-module/transport/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
)

type integrationConfigurationProvider struct {
	configuration configuration.AIConfiguration
}

// GetActive implements configuration.AIConfigurationProvider.
func (p *integrationConfigurationProvider) GetActive(
	ctx context.Context,
) (configuration.AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return configuration.AIConfiguration{}, err
	}

	return p.configuration, nil
}

func (p *integrationConfigurationProvider) GetByRef(
	ctx context.Context,
	ref string,
) (configuration.AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return configuration.AIConfiguration{}, err
	}

	if ref != p.configuration.Ref {
		return configuration.AIConfiguration{}, fmt.Errorf(
			"configuration not found: %s",
			ref,
		)
	}

	return p.configuration, nil
}

type integrationImplementation struct{}

func (i *integrationImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	if err := ctx.Err(); err != nil {
		return executioncontract.AIExecutionResult{}, err
	}

	output, err := anypb.New(
		&aiv1.ExecuteResponse{
			RequestId:  request.RequestID,
			Capability: request.Capability,
		},
	)
	if err != nil {
		return executioncontract.AIExecutionResult{}, err
	}

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     output,
		Metadata: map[string]any{
			"source": "grpc-integration-test",
		},
	}, nil
}

func newIntegrationCredentialResolver() executioncontract.CredentialResolver {
	store := credential.NewMemoryStore()
	service := credential.NewService(store)

	return execution.NewCredentialServiceResolver(service)
}

func TestGRPCServerEndToEnd(t *testing.T) {
	ctx := context.Background()

	// ------------------------------------------------------------
	// 1. Create credential service.
	// ------------------------------------------------------------

	store := credential.NewMemoryStore()
	credentialService := credential.NewService(store)

	credentialRef, err := credentialService.Create(
		ctx,
		[]byte("grpc-test-api-key"),
	)
	if err != nil {
		t.Fatalf(
			"credentialService.Create() failed: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 2. Create configuration provider.
	// ------------------------------------------------------------

	configProvider := &integrationConfigurationProvider{
		configuration: configuration.AIConfiguration{
			Ref:               "config_grpc_test",
			Name:              "grpc-test",
			Type:              "test",
			ImplementationRef: "test.impl",
			CredentialRef:     credentialRef,
			Enabled:           true,
		},
	}

	// ------------------------------------------------------------
	// 3. Create implementation resolver.
	// ------------------------------------------------------------

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		configProvider.configuration.ImplementationRef,
		&integrationImplementation{},
	); err != nil {
		t.Fatalf(
			"resolver.Register() failed: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 4. Create credential resolver.
	// ------------------------------------------------------------

	credentialResolver := execution.NewCredentialServiceResolver(
		credentialService,
	)

	// ------------------------------------------------------------
	// 5. Create executor.
	// ------------------------------------------------------------

	selectionStore := selection.NewMemoryStore()
	selectionService := selection.NewService(selectionStore)

	if err := selectionService.Select(
		ctx,
		configProvider.configuration.Ref,
	); err != nil {
		t.Fatalf(
			"selectionService.Select() failed: %v",
			err,
		)
	}

	executor := execution.NewExecutor(
		configProvider,
		selectionService,
		resolver,
		credentialResolver,
	)

	// ------------------------------------------------------------
	// 6. Start gRPC server.
	// ------------------------------------------------------------

	grpcServer, err := NewGRPCServer(
		executor,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf(
			"NewGRPCServer() failed: %v",
			err,
		)
	}

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- grpcServer.Serve()
	}()

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Second,
		)
		defer cancel()

		_ = grpcServer.Shutdown(shutdownCtx)
	}()

	// ------------------------------------------------------------
	// 7. Connect as external client.
	// ------------------------------------------------------------

	conn, err := grpc.NewClient(
		grpcServer.Address(),
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		t.Fatalf(
			"grpc.NewClient() failed: %v",
			err,
		)
	}

	defer conn.Close()

	client := aiv1.NewAIServiceClient(conn)

	// ------------------------------------------------------------
	// 8. Create request.
	// ------------------------------------------------------------

	input := &anypb.Any{
		TypeUrl: "pride.test/Input",
		Value:   []byte{1, 2, 3, 4},
	}

	request := &aiv1.ExecuteRequest{
		RequestId:  "grpc-e2e-001",
		Capability: "text.generate",
		Input:      input,
	}

	// ------------------------------------------------------------
	// 9. Execute through real gRPC boundary.
	// ------------------------------------------------------------

	response, err := client.Execute(
		ctx,
		request,
	)
	if err != nil {
		t.Fatalf(
			"client.Execute() failed: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 10. Validate response.
	// ------------------------------------------------------------

	if response == nil {
		t.Fatal("response must not be nil")
	}

	if response.GetRequestId() != request.GetRequestId() {
		t.Fatalf(
			"request ID mismatch: got %q, want %q",
			response.GetRequestId(),
			request.GetRequestId(),
		)
	}

	if response.GetCapability() != request.GetCapability() {
		t.Fatalf(
			"capability mismatch: got %q, want %q",
			response.GetCapability(),
			request.GetCapability(),
		)
	}

	if response.GetOutput() == nil {
		t.Fatal("response output must not be nil")
	}

	if response.GetMetadata() == nil {
		t.Fatal("response metadata must not be nil")
	}

	// ------------------------------------------------------------
	// 11. Validate metadata.
	// ------------------------------------------------------------

	sourceField, ok := response.GetMetadata().
		GetFields()["source"]

	if !ok {
		t.Fatal("response metadata does not contain source")
	}

	source := sourceField.GetStringValue()

	if source != "grpc-integration-test" {
		t.Fatalf(
			"unexpected response metadata source: got %q",
			source,
		)
	}

	// ------------------------------------------------------------
	// 12. Make sure server is still running.
	// ------------------------------------------------------------

	select {
	case err := <-serveErr:
		if err != nil && err != grpc.ErrServerStopped {
			t.Fatalf(
				"grpc server stopped unexpectedly: %v",
				err,
			)
		}

	default:
	}
}

func TestGRPCServerRejectsInvalidAddress(t *testing.T) {
	store := selection.NewMemoryStore()
	selector := selection.NewService(store)

	if err := selector.Select(
		context.Background(),
		"config_test",
	); err != nil {
		t.Fatalf("Select() failed: %v", err)
	}

	executor := execution.NewExecutor(
		&integrationConfigurationProvider{
			configuration: configuration.AIConfiguration{
				Ref: "config_test",
			},
		},
		selector,
		implementation.NewResolver(),
		newIntegrationCredentialResolver(),
	)

	_, err := NewGRPCServer(
		executor,
		"",
	)

	if err == nil {
		t.Fatal(
			"expected empty server address to return an error",
		)
	}
}

func TestGRPCServerRejectsNilExecutor(t *testing.T) {
	_, err := NewGRPCServer(
		nil,
		"127.0.0.1:0",
	)

	if err == nil {
		t.Fatal(
			"expected nil executor to return an error",
		)
	}
}

func TestGRPCServerAddress(t *testing.T) {
	store := selection.NewMemoryStore()
	selector := selection.NewService(store)

	if err := selector.Select(
		context.Background(),
		"config_test",
	); err != nil {
		t.Fatalf("Select() failed: %v", err)
	}

	executor := execution.NewExecutor(
		&integrationConfigurationProvider{
			configuration: configuration.AIConfiguration{
				Ref: "config_test",
			},
		},
		selector,
		implementation.NewResolver(),
		newIntegrationCredentialResolver(),
	)

	server, err := NewGRPCServer(
		executor,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf(
			"NewGRPCServer() failed: %v",
			err,
		)
	}

	defer server.Stop()

	address := server.Address()

	if address == "" {
		t.Fatal("server address must not be empty")
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatalf(
			"invalid server address %q: %v",
			address,
			err,
		)
	}

	if host != "127.0.0.1" {
		t.Fatalf(
			"unexpected server host: got %q",
			host,
		)
	}
}

func TestGRPCServerShutdown(t *testing.T) {
	store := selection.NewMemoryStore()
	selector := selection.NewService(store)

	if err := selector.Select(
		context.Background(),
		"config_test",
	); err != nil {
		t.Fatalf("Select() failed: %v", err)
	}

	executor := execution.NewExecutor(
		&integrationConfigurationProvider{
			configuration: configuration.AIConfiguration{
				Ref: "config_test",
			},
		},
		selector,
		implementation.NewResolver(),
		newIntegrationCredentialResolver(),
	)

	server, err := NewGRPCServer(
		executor,
		"127.0.0.1:0",
	)
	if err != nil {
		t.Fatalf(
			"NewGRPCServer() failed: %v",
			err,
		)
	}

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- server.Serve()
	}()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Second,
	)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf(
			"Shutdown() failed: %v",
			err,
		)
	}

	select {
	case err := <-serveErr:
		if err != nil && err != grpc.ErrServerStopped {
			t.Fatalf(
				"Serve() returned unexpected error: %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal(
			"Serve() did not stop after Shutdown()",
		)
	}
}

var _ aiv1.AIServiceServer = (*grpcadapter.Server)(nil)
