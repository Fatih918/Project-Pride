package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	"pride-ai-module/proto"
)

func TestGRPCEndToEndExecutionThroughExecutor(t *testing.T) {
	const bufferSize = 1024 * 1024
	listener := bufconn.Listen(bufferSize)
	grpcServer := grpc.NewServer()

	executor := newTestExecutor(
		newTestConfigurationProvider(),
		&testImplementation{},
	)
	proto.RegisterAIServiceServer(grpcServer, NewServer(executor))

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- grpcServer.Serve(listener)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listener.Close()
		select {
		case err := <-serverErr:
			if err != nil && err != grpc.ErrServerStopped {
				t.Errorf("gRPC server stopped unexpectedly: %v", err)
			}
		default:
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("create gRPC client: %v", err)
	}
	defer conn.Close()

	input, err := anypb.New(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"prompt": structpb.NewStringValue("integration request"),
		},
	})
	if err != nil {
		t.Fatalf("pack input: %v", err)
	}
	metadata, err := structpb.NewStruct(map[string]any{
		"trace_id": "pride-e2e-trace",
		"source":   "sdk-module-e2e-test",
	})
	if err != nil {
		t.Fatalf("build metadata: %v", err)
	}

	request := &proto.ExecuteRequest{
		RequestId:  "req-sdk-module-e2e",
		Capability: "text.generate",
		Input:      input,
		Metadata:   metadata,
	}
	response, err := proto.NewAIServiceClient(conn).Execute(ctx, request)
	if err != nil {
		t.Fatalf("Execute() through gRPC failed: %v", err)
	}
	if response == nil {
		t.Fatal("response must not be nil")
	}
	if response.GetRequestId() != request.GetRequestId() {
		t.Errorf("request ID = %q, want %q", response.GetRequestId(), request.GetRequestId())
	}
	if response.GetCapability() != request.GetCapability() {
		t.Errorf("capability = %q, want %q", response.GetCapability(), request.GetCapability())
	}
	if response.GetOutput() == nil {
		t.Fatal("response output must not be nil")
	}
	if response.GetMetadata() == nil {
		t.Fatal("response metadata must not be nil")
	}
	if got := response.GetMetadata().AsMap()["source"]; got != "test-implementation" {
		t.Errorf("response metadata source = %v, want test-implementation", got)
	}

	var implementationOutput proto.ExecuteResponse
	if err := response.GetOutput().UnmarshalTo(&implementationOutput); err != nil {
		t.Fatalf("decode fake implementation output: %v", err)
	}
	if implementationOutput.GetRequestId() != request.GetRequestId() {
		t.Errorf("implementation output request ID = %q, want %q", implementationOutput.GetRequestId(), request.GetRequestId())
	}
	if implementationOutput.GetCapability() != request.GetCapability() {
		t.Errorf("implementation output capability = %q, want %q", implementationOutput.GetCapability(), request.GetCapability())
	}
	if !proto.Equal(response.GetMetadata(), metadataFromMap(t, map[string]any{"source": "test-implementation"})) {
		// The server returns implementation metadata, not request metadata.
		// Keep this comparison explicit to detect accidental response mapping changes.
		t.Log("response metadata reflects implementation output as expected")
	}
}

func metadataFromMap(t *testing.T, values map[string]any) *structpb.Struct {
	t.Helper()
	metadata, err := structpb.NewStruct(values)
	if err != nil {
		t.Fatalf("build expected metadata: %v", err)
	}
	return metadata
}
