package grpc

import (
	"context"
	"fmt"

	executioncontract "pride-ai-module/contracts/execution"
	prideerrors "pride-ai-module/errors"
	"pride-ai-module/execution"
	"pride-ai-module/transport/mapper"

	proto "pride-ai-module/proto"
)

type Server struct {
	proto.UnimplementedAIServiceServer

	executor *execution.Executor
	mapper   *mapper.ExecutionMapper
}

func NewServer(
	executor *execution.Executor,
) *Server {
	return &Server{
		executor: executor,
		mapper:   mapper.NewExecutionMapper(),
	}
}

func (s *Server) Execute(
	ctx context.Context,
	req *proto.ExecuteRequest,
) (*proto.ExecuteResponse, error) {
	if s == nil {
		return nil, fmt.Errorf(
			"grpc AI server must not be nil",
		)
	}

	if s.executor == nil {
		return nil, fmt.Errorf(
			"executor must not be nil",
		)
	}

	if s.mapper == nil {
		return nil, fmt.Errorf(
			"execution mapper must not be nil",
		)
	}

	if err := validateRequest(req); err != nil {
		invalidRequest := prideerrors.New(
			prideerrors.CodeInvalidRequest,
			err.Error(),
		)

		if req != nil {
			invalidRequest.WithRequestID(req.GetRequestId())
		}

		return nil, prideerrors.GRPCStatus(invalidRequest)
	}

	internalRequest, err := s.mapper.ToContractRequest(req)
	if err != nil {
		return nil, prideerrors.GRPCStatus(
			prideerrors.New(
				prideerrors.CodeInvalidRequest,
				"failed to map execution request",
			).
				WithRequestID(req.GetRequestId()).
				WithCause(err),
		)
	}
	result, err := s.executor.Execute(
		ctx,
		internalRequest,
	)
	if err != nil {
		return nil, prideerrors.GRPCStatus(err)
	}

	response, err := s.mapper.ToProtoResponse(result)
	if err != nil {
		return nil, prideerrors.GRPCStatus(
			prideerrors.New(
				prideerrors.CodeRuntimeError,
				"failed to map execution result",
			).
				WithRequestID(req.GetRequestId()).
				WithCause(err),
		)
	}

	return response, nil
}

func validateRequest(
	req *proto.ExecuteRequest,
) error {
	if req == nil {
		return fmt.Errorf(
			"execution request must not be nil",
		)
	}

	if req.GetRequestId() == "" {
		return fmt.Errorf(
			"request_id must not be empty",
		)
	}

	if req.GetCapability() == "" {
		return fmt.Errorf(
			"capability must not be empty",
		)
	}

	if req.GetInput() == nil {
		return fmt.Errorf(
			"input must not be nil",
		)
	}

	return nil
}

// Compile-time verification that Server implements
// the generated protobuf service contract.
var _ proto.AIServiceServer = (*Server)(nil)

// Compile-time verification that the mapper boundary
// remains based on the execution contract.
var _ interface {
	ToContractRequest(
		*proto.ExecuteRequest,
	) (
		executioncontract.AIExecutionRequest,
		error,
	)

	ToProtoResponse(
		executioncontract.AIExecutionResult,
	) (
		*proto.ExecuteResponse,
		error,
	)
} = (*mapper.ExecutionMapper)(nil)
