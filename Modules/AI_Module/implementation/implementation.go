package implementation

import (
	"context"

	"pride-ai-module/contracts/execution"
)

// AIImplementation is the adapter boundary between PRIDE execution
// and a configured AI provider/runtime.
//
// The implementation is responsible for translating the normalized
// PRIDE execution contract into the provider-facing contract and
// normalizing the provider response back into PRIDE execution.
//
// It may consume:
//   - AIConfiguration
//   - resolved credential
//   - execution request
//
// It must not:
//   - perform configuration selection
//   - access credential storage directly
//   - own configuration lifecycle
//   - expose provider-specific details through the execution contract
type AIImplementation interface {
	Execute(
		ctx context.Context,
		request execution.AIExecutionRequest,
		executionContext execution.AIExecutionContext,
	) (
		execution.AIExecutionResult,
		error,
	)
}
