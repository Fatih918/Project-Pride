package mapper

import (
	"testing"

	executioncontract "pride-ai-module/contracts/execution"
	aiv1 "pride-ai-module/proto"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestExecutionMapperToContractRequest(t *testing.T) {
	mapper := NewExecutionMapper()

	input, err := anypb.New(
		&structpb.Struct{
			Fields: map[string]*structpb.Value{
				"prompt": structpb.NewStringValue("hello"),
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create input Any: %v",
			err,
		)
	}

	request := &aiv1.ExecuteRequest{
		RequestId:  "req-001",
		Capability: "text.generate",
		Input:      input,
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"source": structpb.NewStringValue("test"),
				"retry":  structpb.NewNumberValue(2),
			},
		},
	}

	result, err := mapper.ToContractRequest(request)
	if err != nil {
		t.Fatalf(
			"ToContractRequest() failed: %v",
			err,
		)
	}

	if result.RequestID != request.RequestId {
		t.Fatalf(
			"unexpected RequestID: got %q, want %q",
			result.RequestID,
			request.RequestId,
		)
	}

	if result.Capability != request.Capability {
		t.Fatalf(
			"unexpected Capability: got %q, want %q",
			result.Capability,
			request.Capability,
		)
	}

	if result.Input != input {
		t.Fatal(
			"Input Any was not preserved",
		)
	}

	if result.Metadata["source"] != "test" {
		t.Fatalf(
			"unexpected metadata source: got %v",
			result.Metadata["source"],
		)
	}

	if result.Metadata["retry"] != float64(2) {
		t.Fatalf(
			"unexpected metadata retry: got %v",
			result.Metadata["retry"],
		)
	}
}

func TestExecutionMapperToContractRequestNil(t *testing.T) {
	mapper := NewExecutionMapper()

	_, err := mapper.ToContractRequest(nil)
	if err == nil {
		t.Fatal(
			"expected nil ExecuteRequest to return an error",
		)
	}
}

func TestExecutionMapperToContractRequestNilMetadata(t *testing.T) {
	mapper := NewExecutionMapper()

	request := &aiv1.ExecuteRequest{
		RequestId:  "req-002",
		Capability: "text.generate",
		Input:      nil,
		Metadata:   nil,
	}

	result, err := mapper.ToContractRequest(request)
	if err != nil {
		t.Fatalf(
			"ToContractRequest() failed: %v",
			err,
		)
	}

	if result.RequestID != "req-002" {
		t.Fatalf(
			"unexpected RequestID: got %q",
			result.RequestID,
		)
	}

	if result.Capability != "text.generate" {
		t.Fatalf(
			"unexpected Capability: got %q",
			result.Capability,
		)
	}

	if result.Input != nil {
		t.Fatal(
			"expected nil Input",
		)
	}

	if result.Metadata != nil {
		t.Fatal(
			"expected nil Metadata",
		)
	}
}

func TestExecutionMapperToProtoResponse(t *testing.T) {
	mapper := NewExecutionMapper()

	output, err := anypb.New(
		&structpb.Struct{
			Fields: map[string]*structpb.Value{
				"text": structpb.NewStringValue("generated"),
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"failed to create output Any: %v",
			err,
		)
	}

	result := executioncontract.AIExecutionResult{
		RequestID:  "req-003",
		Capability: "text.generate",
		Output:     output,
		Metadata: map[string]any{
			"provider": "test",
			"tokens":   float64(10),
		},
	}

	response, err := mapper.ToProtoResponse(result)
	if err != nil {
		t.Fatalf(
			"ToProtoResponse() failed: %v",
			err,
		)
	}

	if response == nil {
		t.Fatal(
			"expected non-nil ExecuteResponse",
		)
	}

	if response.RequestId != result.RequestID {
		t.Fatalf(
			"unexpected RequestId: got %q, want %q",
			response.RequestId,
			result.RequestID,
		)
	}

	if response.Capability != result.Capability {
		t.Fatalf(
			"unexpected Capability: got %q, want %q",
			response.Capability,
			result.Capability,
		)
	}

	if response.Output != output {
		t.Fatal(
			"Output Any was not preserved",
		)
	}

	if response.Metadata == nil {
		t.Fatal(
			"expected response metadata",
		)
	}

	metadata := response.Metadata.AsMap()

	if metadata["provider"] != "test" {
		t.Fatalf(
			"unexpected provider metadata: got %v",
			metadata["provider"],
		)
	}

	if metadata["tokens"] != float64(10) {
		t.Fatalf(
			"unexpected tokens metadata: got %v",
			metadata["tokens"],
		)
	}
}

func TestExecutionMapperToProtoResponseNilMetadata(t *testing.T) {
	mapper := NewExecutionMapper()

	result := executioncontract.AIExecutionResult{
		RequestID:  "req-004",
		Capability: "text.generate",
		Output:     nil,
		Metadata:   nil,
	}

	response, err := mapper.ToProtoResponse(result)
	if err != nil {
		t.Fatalf(
			"ToProtoResponse() failed: %v",
			err,
		)
	}

	if response == nil {
		t.Fatal(
			"expected non-nil response",
		)
	}

	if response.RequestId != "req-004" {
		t.Fatalf(
			"unexpected RequestId: got %q",
			response.RequestId,
		)
	}

	if response.Capability != "text.generate" {
		t.Fatalf(
			"unexpected Capability: got %q",
			response.Capability,
		)
	}

	if response.Output != nil {
		t.Fatal(
			"expected nil Output",
		)
	}

	if response.Metadata != nil {
		t.Fatal(
			"expected nil Metadata",
		)
	}
}

func TestExecutionMapperRejectsInvalidMetadata(t *testing.T) {
	mapper := NewExecutionMapper()

	result := executioncontract.AIExecutionResult{
		RequestID:  "req-005",
		Capability: "text.generate",
		Metadata: map[string]any{
			"unsupported": func() {},
		},
	}

	_, err := mapper.ToProtoResponse(result)
	if err == nil {
		t.Fatal(
			"expected unsupported metadata value to return an error",
		)
	}
}
