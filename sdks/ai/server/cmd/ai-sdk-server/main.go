package main

import (
	"log"
	"net"
	"os"

	aiv1 "pride-ai-sdk/proto"
	"pride-ai-sdk/server/internal/execution"
	aiservice "pride-ai-sdk/server/internal/service/ai"
	grpcclient "pride-ai-sdk/server/internal/transport/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultSDKServerAddress = ":50051"
	defaultAIModuleAddress  = "localhost:50052"
)

func main() {
	// ------------------------------------------------------------
	// 1. Resolve AI Module address
	// ------------------------------------------------------------

	aiModuleAddress := os.Getenv("PRIDE_AI_MODULE_ADDR")
	if aiModuleAddress == "" {
		aiModuleAddress = defaultAIModuleAddress
	}

	// ------------------------------------------------------------
	// 2. Resolve SDK server address
	// ------------------------------------------------------------

	sdkServerAddress := os.Getenv("PRIDE_AI_SDK_ADDR")
	if sdkServerAddress == "" {
		sdkServerAddress = defaultSDKServerAddress
	}

	// ------------------------------------------------------------
	// 3. Connect SDK → AI Module
	// ------------------------------------------------------------

	moduleConn, err := grpc.NewClient(
		aiModuleAddress,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		log.Fatalf(
			"failed to create AI Module connection: %v",
			err,
		)
	}

	defer moduleConn.Close()

	// ------------------------------------------------------------
	// 4. Create AI Module client
	// ------------------------------------------------------------

	aiModuleClient := aiv1.NewAIServiceClient(
		moduleConn,
	)

	// ------------------------------------------------------------
	// 5. Create SDK gRPC client adapter
	// ------------------------------------------------------------

	aiClient := grpcclient.NewGRPCAIClient(
		aiModuleClient,
	)

	// ------------------------------------------------------------
	// 6. Create SDK Executor
	// ------------------------------------------------------------

	executor := execution.NewExecutor(
		aiClient,
	)

	// ------------------------------------------------------------
	// 7. Create SDK AI Service
	// ------------------------------------------------------------

	aiService := aiservice.NewService(
		executor,
	)

	// ------------------------------------------------------------
	// 8. Listen for external SDK clients
	// ------------------------------------------------------------

	listener, err := net.Listen(
		"tcp",
		sdkServerAddress,
	)
	if err != nil {
		log.Fatalf(
			"failed to listen on %s: %v",
			sdkServerAddress,
			err,
		)
	}

	// ------------------------------------------------------------
	// 9. Create gRPC server
	// ------------------------------------------------------------

	grpcServer := grpc.NewServer()

	aiv1.RegisterAIServiceServer(
		grpcServer,
		aiService,
	)

	// ------------------------------------------------------------
	// 10. Start SDK server
	// ------------------------------------------------------------

	log.Printf(
		"PRIDE AI SDK Server listening on %s",
		sdkServerAddress,
	)

	log.Printf(
		"PRIDE AI SDK connected to AI Module at %s",
		aiModuleAddress,
	)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf(
			"gRPC server stopped: %v",
			err,
		)
	}
}
