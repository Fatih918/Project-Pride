package module

import (
	"context"

	executioncontract "pride-ai-sdk/server/internal/contracts/execution"
)

// AIClient defines the communication boundary between
// the AI SDK and the AI Module.
//
// The SDK depends only on this contract.
// The transport implementation (gRPC) remains outside
// the SDK execution domain.
type AIClient interface {
	Execute(
		ctx context.Context,
		request executioncontract.AIExecutionRequest,
	) (executioncontract.AIExecutionResult, error)
}
