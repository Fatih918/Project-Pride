package main

import (
	"context"
	"log"
	"time"

	aiv1 "pride-ai-sdk/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func main() {
	conn, err := grpc.NewClient(
		"localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := aiv1.NewAIServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inputStruct, err := structpb.NewStruct(map[string]any{
		"message": "hello PRIDE",
		"value":   42,
	})
	if err != nil {
		log.Fatalf("failed to create input struct: %v", err)
	}

	inputAny, err := anypb.New(inputStruct)
	if err != nil {
		log.Fatalf("failed to create input Any: %v", err)
	}

	metadata := structpb.NewStructValue(
		&structpb.Struct{
			Fields: map[string]*structpb.Value{
				"source": structpb.NewStringValue("integration-test"),
			},
		},
	)

	response, err := client.Execute(ctx, &aiv1.ExecuteRequest{
		RequestId:  "test-request-002",
		Capability: "test",
		Input:      inputAny,
		Metadata:   metadata.GetStructValue(),
	})
	if err != nil {
		log.Fatalf("Execute failed: %v", err)
	}

	log.Printf("Request ID: %s", response.GetRequestId())
	log.Printf("Capability: %s", response.GetCapability())
	log.Printf("Output Type: %s", response.GetOutput().GetTypeUrl())
	log.Printf("Metadata: %v", response.GetMetadata().AsMap())
}
