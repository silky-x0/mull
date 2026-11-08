// Package provider defines how mull talks to an AI backend.
package provider

import "context"

// Provider sends an instruction plus input text to a model and returns its reply.
// Every backend (Claude CLI, Gemini CLI, OpenRouter, Ollama...) implements this.
type Provider interface {
	Complete(ctx context.Context, instruction, input string) (Reply, error)
}

type Reply struct {
	Text string
	// Model is the one that actually answered; it can differ from the
	// requested one (e.g. openrouter/free picks a model per request).
	// Empty if the backend doesn't report it.
	Model string
}
