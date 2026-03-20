// Package agent defines the Agent interface and shared review types used
// across all consul implementations (binary, API, etc.).
package agent

import "context"

// Agent is the contract every consul implementation must satisfy.
// The orchestrator calls Review() — it never knows whether the implementation
// shells out to a local binary or hits a cloud API.
type Agent interface {
	// Name returns the consul's identifier (e.g. "gemini", "copilot", "oz").
	Name() string

	// Review performs the code review for the given request and returns the
	// review output. Implementations are expected to run to completion and
	// return the full review text; partial output is not streamed.
	Review(ctx context.Context, req ReviewRequest) (ReviewResult, error)
}

// ReviewRequest carries all information needed for a single review invocation.
// Binary agents use Prompt directly; future API agents may reconstruct the
// prompt programmatically from the same data.
type ReviewRequest struct {
	// Prompt is the fully-constructed review prompt:
	// skill content + PR title + body + diff, pre-built by the orchestrator.
	Prompt string

	// Model is the model identifier for this consul (e.g. "gemini-2.5-pro").
	Model string

	// ExtraArgs are optional CLI flags for binary agents that override the
	// built-in non-interactive defaults. Ignored by API implementations.
	ExtraArgs []string
}

// ReviewResult holds the outcome of a single agent review.
type ReviewResult struct {
	// AgentName matches Agent.Name().
	AgentName string

	// Output is the full review text returned by the agent.
	Output string
}
