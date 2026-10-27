package grpc_test

import (
	"context"
	stderrors "errors"
	"net"
	"strings"
	"testing"
	"time"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	prideerrors "pride-ai-module/errors"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/selection"
	grpcadapter "pride-ai-module/transport/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	r10ConfigurationRef  = "r10.configuration"
	r10ImplementationRef = "r10.auth-failure"
	r10SecretSentinel    = "api-key=R10_DO_NOT_LEAK_SECRET"
)

type r10AuthenticationFailureImplementation struct{}

func (r10AuthenticationFailureImplementation) Execute(
	context.Context,
	executioncontract.AIExecutionRequest,
	executioncontract.AIExecutionContext,
) (executioncontract.AIExecutionResult, error) {
	return executioncontract.AIExecutionResult{}, prideerrors.New(
		prideerrors.CodeAuthenticationError,
		"AI provider authentication failed",
	).WithCause(stderrors.New(r10SecretSentinel))
}

func TestAuthenticationErrorSurvivesGRPCE2EWithoutLeakingSecret(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configurations := configuration.NewMemoryStore()
	config := configuration.AIConfiguration{
		Ref:               r10ConfigurationRef,
		Name:              "R10 error E2E",
		Type:              "test",
		ImplementationRef: r10ImplementationRef,
		Enabled:           false,
	}
	if err := configurations.Create(ctx, config); err != nil {
		t.Fatalf("create configuration: %v", err)
	}
	if err := configurations.Select(ctx, r10ConfigurationRef); err != nil {
		t.Fatalf("select configuration: %v", err)
	}

	selectionService := selection.NewService(selection.NewMemoryStore())
	if err := selectionService.Select(ctx, r10ConfigurationRef); err != nil {
		t.Fatalf("select configuration reference: %v", err)
	}

	implementations := implementation.NewResolver()
	if err := implementations.Register(
		r10ImplementationRef,
		r10AuthenticationFailureImplementation{},
	); err != nil {
		t.Fatalf("register authentication-failure implementation: %v", err)
	}

	executor := execution.NewExecutor(
		configurations,
		selectionService,
		implementations,
		nil,
	)

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	aiv1.RegisterAIServiceServer(server, grpcadapter.NewServer(executor))
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	connection, err := grpc.DialContext(
		ctx,
		"bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial in-memory gRPC server: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	input, err := anypb.New(wrapperspb.String("r10-auth-failure-payload"))
	if err != nil {
		t.Fatalf("encode input: %v", err)
	}

	_, err = aiv1.NewAIServiceClient(connection).Execute(ctx, &aiv1.ExecuteRequest{
		RequestId:  "r10-auth-request",
		Capability: "text.generate",
		Input:      input,
	})
	if err == nil {
		t.Fatal("expected authentication error")
	}

	grpcStatus := status.Convert(err)
	if grpcStatus.Code() != codes.Unauthenticated {
		t.Fatalf("gRPC code = %v, want %v; error: %v", grpcStatus.Code(), codes.Unauthenticated, err)
	}
	if grpcStatus.Message() != "AI provider authentication failed" {
		t.Fatalf("gRPC message = %q, want safe authentication message", grpcStatus.Message())
	}
	if strings.Contains(err.Error(), r10SecretSentinel) ||
		strings.Contains(grpcStatus.Message(), r10SecretSentinel) {
		t.Fatal("gRPC error exposed the internal secret")
	}

	remoteError, ok := prideerrors.FromGRPCStatus(err)
	if !ok {
		t.Fatal("expected standardized AIError detail in gRPC response")
	}
	if remoteError.Code != prideerrors.CodeAuthenticationError {
		t.Fatalf("remote error code = %q, want %q", remoteError.Code, prideerrors.CodeAuthenticationError)
	}
	if remoteError.Code == prideerrors.CodeUnknownError {
		t.Fatal("authentication error was downgraded to UNKNOWN_ERROR")
	}
	if remoteError.RequestID != "r10-auth-request" {
		t.Fatalf("remote request ID = %q, want r10-auth-request", remoteError.RequestID)
	}
	if remoteError.Message != "AI provider authentication failed" {
		t.Fatalf("remote error message = %q, want safe authentication message", remoteError.Message)
	}
	if strings.Contains(remoteError.Message, r10SecretSentinel) {
		t.Fatal("AIError detail exposed the internal secret")
	}
}
