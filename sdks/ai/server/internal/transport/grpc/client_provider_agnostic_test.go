package grpc

import (
	"context"
	"reflect"
	"testing"

	aiv1 "pride-ai-sdk/proto"
	executioncontract "pride-ai-sdk/server/internal/contracts/execution"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type providerAgnosticServiceClient struct {
	request  *aiv1.ExecuteRequest
	response *aiv1.ExecuteResponse
	ctxValue any
	err      error
}

type sdkContextKey struct{}

func (c *providerAgnosticServiceClient) Execute(
	ctx context.Context,
	request *aiv1.ExecuteRequest,
	_ ...grpc.CallOption,
) (*aiv1.ExecuteResponse, error) {
	c.request = request
	c.ctxValue = ctx.Value(sdkContextKey{})
	return c.response, c.err
}

func TestGRPCAIClientForwardsProviderAgnosticContract(t *testing.T) {
	input, err := anypb.New(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"payload": structpb.NewStringValue("opaque input"),
		},
	})
	if err != nil {
		t.Fatalf("pack input: %v", err)
	}

	output, err := anypb.New(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"result": structpb.NewStringValue("opaque output"),
		},
	})
	if err != nil {
		t.Fatalf("pack output: %v", err)
	}

	requestMetadata := map[string]any{
		"trace_id": "trace-sdk-agnostic",
		"options": map[string]any{
			"temperature": 0.25,
		},
	}
	responseMetadata := map[string]any{
		"trace_id": "trace-sdk-agnostic",
		"execution": map[string]any{
			"attempt": float64(1),
		},
	}
	metadataProto, err := structpb.NewStruct(responseMetadata)
	if err != nil {
		t.Fatalf("build response metadata: %v", err)
	}

	fake := &providerAgnosticServiceClient{
		response: &aiv1.ExecuteResponse{
			RequestId:  "req-sdk-agnostic",
			Capability: "custom.vendor.operation",
			Output:     output,
			Metadata:   metadataProto,
		},
	}
	client := NewGRPCAIClient(fake)
	ctx := context.WithValue(context.Background(), sdkContextKey{}, "context-preserved")

	got, err := client.Execute(ctx, executioncontract.AIExecutionRequest{
		RequestID:  "req-sdk-agnostic",
		Capability: "custom.vendor.operation",
		Input:      input,
		Metadata:   requestMetadata,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if fake.request == nil {
		t.Fatal("AI Module was not called")
	}
	if fake.request.GetRequestId() != "req-sdk-agnostic" {
		t.Errorf("request ID = %q, want %q", fake.request.GetRequestId(), "req-sdk-agnostic")
	}
	if fake.request.GetCapability() != "custom.vendor.operation" {
		t.Errorf("capability = %q, want custom.vendor.operation", fake.request.GetCapability())
	}
	if !proto.Equal(fake.request.GetInput(), input) {
		t.Error("input Any payload was changed in transport")
	}
	if gotMetadata := fake.request.GetMetadata().AsMap(); !reflect.DeepEqual(gotMetadata, requestMetadata) {
		t.Errorf("request metadata = %#v, want %#v", gotMetadata, requestMetadata)
	}
	if fake.ctxValue != "context-preserved" {
		t.Errorf("context value = %#v, want %q", fake.ctxValue, "context-preserved")
	}

	if got.RequestID != "req-sdk-agnostic" {
		t.Errorf("response request ID = %q, want %q", got.RequestID, "req-sdk-agnostic")
	}
	if got.Capability != "custom.vendor.operation" {
		t.Errorf("response capability = %q, want custom.vendor.operation", got.Capability)
	}
	if !proto.Equal(got.Output, output) {
		t.Error("output Any payload was changed in transport")
	}
	if !reflect.DeepEqual(got.Metadata, responseMetadata) {
		t.Errorf("response metadata = %#v, want %#v", got.Metadata, responseMetadata)
	}
}
