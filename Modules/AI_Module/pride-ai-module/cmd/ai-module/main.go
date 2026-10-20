package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	"pride-ai-module/selection"
	"pride-ai-module/server"

	aiv1 "pride-ai-module/proto"

	"google.golang.org/protobuf/types/known/anypb"
)

const (
	defaultServerAddress = ":50052"

	configurationRef  = "config_integration"
	implementationRef = "integration.echo"
)

// integrationImplementation is a minimal implementation used by
// the standalone AI Module executable for SDK ↔ Module integration testing.
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

	outputMessage := &aiv1.ExecuteResponse{
		RequestId:  request.RequestID,
		Capability: request.Capability,
		Output:     request.Input,
	}

	output, err := anypb.New(outputMessage)
	if err != nil {
		return executioncontract.AIExecutionResult{}, err
	}

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     output,
		Metadata: map[string]any{
			"source": "ai-module",
		},
	}, nil
}

func main() {
	// ------------------------------------------------------------
	// 1. Shutdown context
	// ------------------------------------------------------------

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// ------------------------------------------------------------
	// 2. Server address
	// ------------------------------------------------------------

	address := os.Getenv("PRIDE_AI_MODULE_ADDR")
	if address == "" {
		address = defaultServerAddress
	}

	// ------------------------------------------------------------
	// 3. Credential
	// ------------------------------------------------------------

	credentialStore := credential.NewMemoryStore()

	credentialService := credential.NewService(
		credentialStore,
	)

	credentialRef, err := credentialService.Create(
		ctx,
		[]byte("integration-test-api-key"),
	)
	if err != nil {
		log.Fatalf(
			"failed to create integration credential: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 4. Configuration
	// ------------------------------------------------------------

	configurationStore := configuration.NewMemoryStore()

	aiConfiguration := configuration.AIConfiguration{
		Ref:               configurationRef,
		Name:              "integration-configuration",
		Type:              "test",
		ImplementationRef: implementationRef,
		CredentialRef:     credentialRef,
		Enabled:           false,
	}

	if err := configurationStore.Create(
		ctx,
		aiConfiguration,
	); err != nil {
		log.Fatalf(
			"failed to create AI configuration: %v",
			err,
		)
	}

	if err := configurationStore.Select(
		ctx,
		configurationRef,
	); err != nil {
		log.Fatalf(
			"failed to select AI configuration: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 5. Selection
	//
	// Selection stores ONLY ConfigurationRef.
	// ------------------------------------------------------------

	selectionStore := selection.NewMemoryStore()

	selectionService := selection.NewService(
		selectionStore,
	)

	if err := selectionService.Select(
		ctx,
		configurationRef,
	); err != nil {
		log.Fatalf(
			"failed to select AI configuration: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 6. Implementation resolver
	// ------------------------------------------------------------

	expected := &integrationImplementation{}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		implementationRef,
		expected,
	); err != nil {
		log.Fatalf(
			"failed to register AI implementation: %v",
			err,
		)
	}
	// ------------------------------------------------------------
	// 7. Credential resolver
	// ------------------------------------------------------------

	credentialResolver := execution.NewCredentialServiceResolver(
		credentialService,
	)

	// ------------------------------------------------------------
	// 8. Execution
	// ------------------------------------------------------------

	executor := execution.NewExecutor(
		configurationStore,
		selectionService,
		resolver,
		credentialResolver,
	)

	// ------------------------------------------------------------
	// 9. gRPC Server
	// ------------------------------------------------------------

	grpcServer, err := server.NewGRPCServer(
		executor,
		address,
	)
	if err != nil {
		log.Fatalf(
			"failed to create AI Module gRPC server: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 10. Start server
	// ------------------------------------------------------------

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- grpcServer.Serve()
	}()

	log.Printf(
		"PRIDE AI Module listening on %s",
		grpcServer.Address(),
	)

	log.Printf(
		"PRIDE AI Module configuration: %s",
		configurationRef,
	)

	log.Printf(
		"PRIDE AI Module implementation: %s",
		implementationRef,
	)

	// ------------------------------------------------------------
	// 11. Wait for shutdown
	// ------------------------------------------------------------

	select {
	case <-ctx.Done():
		log.Printf("shutdown signal received")

	case err := <-serveErr:
		if err != nil {
			log.Fatalf(
				"AI Module gRPC server stopped: %v",
				err,
			)
		}

		return
	}

	// ------------------------------------------------------------
	// 12. Graceful shutdown
	// ------------------------------------------------------------

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := grpcServer.Shutdown(shutdownCtx); err != nil {
		log.Printf(
			"AI Module graceful shutdown failed: %v",
			err,
		)

		grpcServer.Stop()
		return
	}

	log.Printf("PRIDE AI Module stopped")
}
