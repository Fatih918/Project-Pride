package execution

import "google.golang.org/protobuf/types/known/anypb"

type AIExecutionRequest struct {
	RequestID  string
	Capability string
	Input      *anypb.Any
	Metadata   map[string]any
}
