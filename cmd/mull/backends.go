package main

import (
	"cmp"
	"errors"
	"os"
	"os/exec"

	"github.com/silky-x0/mull/internal/provider"
)

const (
	groqURL         = "https://api.groq.com/openai/v1"
	groqModel       = "openai/gpt-oss-20b"
	openRouterURL   = "https://openrouter.ai/api/v1"
	openRouterModel = "openrouter/free" // routes to whichever free model is available
	claudeModel     = "haiku"           // fast enough for a fallback; the CLI adds ~3s startup
)

// backends lists what to try, in order: an API (Groq, else OpenRouter) for
// speed, then the claude CLI, if installed, on the user's subscription.
func backends() ([]provider.Provider, error) {
	var ps []provider.Provider
	model := os.Getenv("MULL_MODEL")
	if key := os.Getenv("GROQ_API_KEY"); key != "" {
		ps = append(ps, provider.OpenAI{BaseURL: groqURL, APIKey: key, Model: cmp.Or(model, groqModel)})
	} else if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		ps = append(ps, provider.OpenAI{BaseURL: openRouterURL, APIKey: key, Model: cmp.Or(model, openRouterModel)})
	}
	if _, err := exec.LookPath("claude"); err == nil {
		ps = append(ps, provider.ClaudeCLI{Model: claudeModel})
	}
	if len(ps) == 0 {
		return nil, errors.New("no AI backend: set GROQ_API_KEY (https://console.groq.com/keys) or OPENROUTER_API_KEY, or install and log in to the claude CLI")
	}
	return ps, nil
}
