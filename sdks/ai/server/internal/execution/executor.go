package execution

import (
	"context"
	"errors"

	executioncontract "pride-ai-sdk/server/internal/contracts/execution"
	modulecontract "pride-ai-sdk/server/internal/contracts/module"
)

var (
	ErrNilAIClient = errors.New("AI module client is nil")
)

type Executor struct {
	client modulecontract.AIClient
}

func NewExecutor(client modulecontract.AIClient) *Executor {
	return &Executor{
		client: client,
	}
}

func (e *Executor) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
) (executioncontract.AIExecutionResult, error) {
	if e.client == nil {
		return executioncontract.AIExecutionResult{}, ErrNilAIClient
	}

	return e.client.Execute(ctx, request)
}
