// Package provider defines how mull talks to an AI backend.
package provider

import "context"

// Provider sends an instruction plus input text to a model and returns its reply.
// Every backend (Claude CLI, Gemini CLI, OpenRouter, Ollama...) implements this.
type Provider interface {
	Complete(ctx context.Context, instruction, input string) (string, error)
}
