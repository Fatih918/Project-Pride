package execution

import "google.golang.org/protobuf/types/known/anypb"

type AIExecutionResult struct {
	RequestID  string
	Capability string
	Output     *anypb.Any
	Metadata   map[string]any
}
