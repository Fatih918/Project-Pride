package mapper

import (
	"fmt"

	executioncontract "pride-ai-module/contracts/execution"
	aiv1 "pride-ai-module/proto"

	"google.golang.org/protobuf/types/known/structpb"
)

// ExecutionMapper maps transport-level protobuf messages
// to internal execution contracts and vice versa.
//
// Boundary:
//
//	Proto <-> Execution Contract
//
// This mapper must not:
//   - resolve configuration
//   - resolve credentials
//   - resolve implementations
//   - execute AI providers
//   - apply provider-specific logic
type ExecutionMapper struct{}

// NewExecutionMapper creates a new execution mapper.
func NewExecutionMapper() *ExecutionMapper {
	return &ExecutionMapper{}
}

// ToContractRequest converts a protobuf ExecuteRequest
// into the internal AIExecutionRequest contract.
func (m *ExecutionMapper) ToContractRequest(
	request *aiv1.ExecuteRequest,
) (executioncontract.AIExecutionRequest, error) {
	if request == nil {
		return executioncontract.AIExecutionRequest{}, fmt.Errorf(
			"execute request must not be nil",
		)
	}

	metadata, err := structToMap(
		request.GetMetadata(),
	)
	if err != nil {
		return executioncontract.AIExecutionRequest{}, fmt.Errorf(
			"map execution request metadata: %w",
			err,
		)
	}

	return executioncontract.AIExecutionRequest{
		RequestID:  request.GetRequestId(),
		Capability: request.GetCapability(),
		Input:      request.GetInput(),
		Metadata:   metadata,
	}, nil
}

// ToProtoResponse converts an internal AIExecutionResult
// into a protobuf ExecuteResponse.
func (m *ExecutionMapper) ToProtoResponse(
	result executioncontract.AIExecutionResult,
) (*aiv1.ExecuteResponse, error) {
	metadata, err := mapToStruct(
		result.Metadata,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"map execution result metadata: %w",
			err,
		)
	}

	return &aiv1.ExecuteResponse{
		RequestId:  result.RequestID,
		Capability: result.Capability,
		Output:     result.Output,
		Metadata:   metadata,
	}, nil
}

// structToMap converts protobuf Struct metadata
// into the internal map representation.
//
// A nil protobuf Struct is intentionally mapped to nil.
func structToMap(
	value *structpb.Struct,
) (map[string]any, error) {
	if value == nil {
		return nil, nil
	}

	result := value.AsMap()

	if result == nil {
		return nil, nil
	}

	return result, nil
}

// mapToStruct converts internal metadata into protobuf Struct.
//
// A nil map is intentionally mapped to nil.
func mapToStruct(
	value map[string]any,
) (*structpb.Struct, error) {
	if value == nil {
		return nil, nil
	}

	result, err := structpb.NewStruct(value)
	if err != nil {
		return nil, fmt.Errorf(
			"create protobuf Struct: %w",
			err,
		)
	}

	return result, nil
}
