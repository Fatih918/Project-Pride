package grpc_test

import (
	"context"
	"net"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	localadapter "pride-ai-module/implementation/local"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/provider"
	"pride-ai-module/selection"
	grpcadapter "pride-ai-module/transport/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	r9ConfigurationRef  = "r9.configuration"
	r9ImplementationRef = "r9.test-implementation"
	r9ProviderRef       = "r9.test-provider"
)

type r9KeySource struct{}

func (r9KeySource) GetKey(context.Context) ([]byte, error) {
	return []byte("0123456789abcdef0123456789abcdef"), nil
}

type r9TestProvider struct {
	called atomic.Bool
}

func (p *r9TestProvider) Execute(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	if err := ctx.Err(); err != nil {
		return provider.Response{}, err
	}
	p.called.Store(true)
	return provider.Response{
		Output: request.Input,
		Metadata: map[string]any{
			"provider":   "r9-test-provider",
			"capability": request.Capability,
		},
	}, nil
}

type r9CredentialCheckingImplementation struct {
	delegate interface {
		Execute(
			context.Context,
			executioncontract.AIExecutionRequest,
			executioncontract.AIExecutionContext,
		) (executioncontract.AIExecutionResult, error)
	}
}

func (i *r9CredentialCheckingImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (executioncontract.AIExecutionResult, error) {
	if string(executionContext.Credential) != "r9-internal-test-secret" {
		return executioncontract.AIExecutionResult{}, errR9CredentialMismatch
	}
	return i.delegate.Execute(ctx, request, executionContext)
}

type r9TestError string

func (e r9TestError) Error() string { return string(e) }

const errR9CredentialMismatch = r9TestError("resolved credential did not match test secret")

func TestFullInternalE2EUsesPersistentStoresResolversAndGRPC(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configurationDB := filepath.Join(t.TempDir(), "configuration.db")
	configurations, err := configuration.NewSQLiteStore(ctx, configurationDB)
	if err != nil {
		t.Fatalf("create SQLite configuration store: %v", err)
	}
	t.Cleanup(func() { _ = configurations.Close() })

	encryptedCredentials, credentialDB, err := credential.NewSQLiteEncryptedStore(
		ctx,
		filepath.Join(t.TempDir(), "credentials.db"),
		r9KeySource{},
	)
	if err != nil {
		t.Fatalf("create encrypted SQLite credential store: %v", err)
	}
	t.Cleanup(func() { _ = credentialDB.Close() })

	credentialService := credential.NewService(encryptedCredentials)
	credentialRef, err := credentialService.Create(ctx, []byte("r9-internal-test-secret"))
	if err != nil {
		t.Fatalf("create test credential: %v", err)
	}

	// Confirm the real SQLite storage layer holds ciphertext, not the plaintext secret.
	storedCredential, err := credentialDB.Get(ctx, credentialRef)
	if err != nil {
		t.Fatalf("read stored ciphertext: %v", err)
	}
	if strings.Contains(string(storedCredential.Value), "r9-internal-test-secret") {
		t.Fatal("credential store contains plaintext secret")
	}

	configuration := configuration.AIConfiguration{
		Ref:                  r9ConfigurationRef,
		Name:                 "R9 internal E2E",
		Type:                 "test",
		ImplementationRef:    r9ImplementationRef,
		CredentialRef:        credentialRef,
		Enabled:              false,
		ConnectionParameters: map[string]any{"model": "r9-test-model"},
	}
	if err := configurations.Create(ctx, configuration); err != nil {
		t.Fatalf("create AI configuration: %v", err)
	}
	if err := configurations.Select(ctx, r9ConfigurationRef); err != nil {
		t.Fatalf("select AI configuration: %v", err)
	}

	selectionStore := selection.NewMemoryStore()
	selectionService := selection.NewService(selectionStore)
	if err := selectionService.Select(ctx, r9ConfigurationRef); err != nil {
		t.Fatalf("select configuration reference: %v", err)
	}

	providerResolver := provider.NewResolver()
	testProvider := &r9TestProvider{}
	if err := providerResolver.Register(r9ProviderRef, testProvider); err != nil {
		t.Fatalf("register test provider: %v", err)
	}
	resolvedProvider, err := providerResolver.Resolve(r9ProviderRef)
	if err != nil {
		t.Fatalf("resolve test provider: %v", err)
	}
	adapter, err := localadapter.NewAdapter(resolvedProvider)
	if err != nil {
		t.Fatalf("create provider adapter: %v", err)
	}

	implementationResolver := implementation.NewResolver()
	testImplementation := &r9CredentialCheckingImplementation{delegate: adapter}
	if err := implementationResolver.Register(r9ImplementationRef, testImplementation); err != nil {
		t.Fatalf("register test implementation: %v", err)
	}

	credentialResolver := credential.NewResolver(credentialService)
	executor := execution.NewExecutor(
		configurations,
		selectionService,
		implementationResolver,
		credentialResolver,
	)

	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	aiv1.RegisterAIServiceServer(grpcServer, grpcadapter.NewServer(executor))
	go func() {
		_ = grpcServer.Serve(listener)
	}()
	t.Cleanup(grpcServer.Stop)

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

	input, err := anypb.New(wrapperspb.String("r9-payload"))
	if err != nil {
		t.Fatalf("encode input: %v", err)
	}
	metadata, err := structpb.NewStruct(map[string]any{
		"trace":   map[string]any{"source": "r9-test"},
		"attempt": float64(1),
	})
	if err != nil {
		t.Fatalf("encode metadata: %v", err)
	}

	response, err := aiv1.NewAIServiceClient(connection).Execute(ctx, &aiv1.ExecuteRequest{
		RequestId:  "r9-request",
		Capability: "text.generate",
		Input:      input,
		Metadata:   metadata,
	})
	if err != nil {
		t.Fatalf("execute full internal E2E request: %v", err)
	}

	if response.GetRequestId() != "r9-request" {
		t.Fatalf("request ID = %q, want r9-request", response.GetRequestId())
	}
	if response.GetCapability() != "text.generate" {
		t.Fatalf("capability = %q, want text.generate", response.GetCapability())
	}
	if !proto.Equal(response.GetOutput(), input) {
		t.Fatal("response output did not preserve the input Any payload")
	}
	if response.GetMetadata().GetFields()["provider"].GetStringValue() != "r9-test-provider" {
		t.Fatalf("unexpected provider metadata: %v", response.GetMetadata().AsMap())
	}
	if response.GetMetadata().GetFields()["capability"].GetStringValue() != "text.generate" {
		t.Fatalf("unexpected capability metadata: %v", response.GetMetadata().AsMap())
	}
	if !testProvider.called.Load() {
		t.Fatal("test provider was not reached through the provider resolver and adapter")
	}
}
