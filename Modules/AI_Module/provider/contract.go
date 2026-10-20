package provider

import "context"

// Request represents a provider-agnostic request
// passed from an implementation to a provider.
type Request struct {
	Capability string
	Input      any
	Metadata   map[string]any
}

// Response represents a provider-agnostic response
// returned by a provider.
type Response struct {
	Output   any
	Metadata map[string]any
}

// Provider defines the boundary between an AI implementation
// and an external AI provider.
//
// Provider implementations are responsible for translating
// this provider-agnostic contract into provider-specific
// API requests and responses.
type Provider interface {
	Execute(
		ctx context.Context,
		request Request,
	) (Response, error)
}

// Compile-time documentation:
//
// Provider must remain independent from:
//   - Selection
//   - Configuration storage
//   - Credential storage
//   - Execution orchestration
//   - Provider-specific SDK types
