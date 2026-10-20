package ai

import (
	"context"
	"errors"

	aiv1 "pride-ai-sdk/proto"
	executioncontract "pride-ai-sdk/server/internal/contracts/execution"
	"pride-ai-sdk/server/internal/execution"

	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ErrInvalidRequest = errors.New("invalid AI execution request")
)

type Service struct {
	aiv1.UnimplementedAIServiceServer

	executor *execution.Executor
}

func NewService(executor *execution.Executor) *Service {
	return &Service{
		executor: executor,
	}
}

func (s *Service) Execute(
	ctx context.Context,
	req *aiv1.ExecuteRequest,
) (*aiv1.ExecuteResponse, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	if s.executor == nil {
		return nil, execution.ErrNilAIClient
	}

	result, err := s.executor.Execute(
		ctx,
		executioncontract.AIExecutionRequest{
			RequestID:  req.GetRequestId(),
			Capability: req.GetCapability(),
			Input:      req.GetInput(),
			Metadata:   structToMap(req.GetMetadata()),
		},
	)
	if err != nil {
		return nil, err
	}

	return &aiv1.ExecuteResponse{
		RequestId:  result.RequestID,
		Capability: result.Capability,
		Output:     result.Output,
		Metadata:   mapToStruct(result.Metadata),
	}, nil
}

func validateRequest(req *aiv1.ExecuteRequest) error {
	if req == nil {
		return ErrInvalidRequest
	}

	if req.GetRequestId() == "" {
		return ErrInvalidRequest
	}

	if req.GetCapability() == "" {
		return ErrInvalidRequest
	}

	if req.GetInput() == nil {
		return ErrInvalidRequest
	}

	return nil
}

func structToMap(value *structpb.Struct) map[string]any {
	if value == nil {
		return nil
	}

	return value.AsMap()
}

func mapToStruct(value map[string]any) *structpb.Struct {
	if value == nil {
		return nil
	}

	result, err := structpb.NewStruct(value)
	if err != nil {
		return nil
	}

	return result
}
