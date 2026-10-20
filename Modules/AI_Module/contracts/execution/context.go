package execution

import "pride-ai-module/configuration"

type AIExecutionContext struct {
	Configuration configuration.AIConfiguration
	Credential    []byte
}
