package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/silky-x0/mull/internal/provider"
)

func TestBackends(t *testing.T) {
	withClaude := t.TempDir()
	if err := os.WriteFile(filepath.Join(withClaude, "claude"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	noClaude := t.TempDir()

	groq := provider.OpenAI{BaseURL: groqURL, APIKey: "g", Model: groqModel}
	or := provider.OpenAI{BaseURL: openRouterURL, APIKey: "o", Model: openRouterModel}
	claude := provider.ClaudeCLI{Model: claudeModel}

	tests := []struct {
		name             string
		groq, openrouter string
		model, path      string
		want             []provider.Provider
	}{
		{"groq then claude", "g", "", "", withClaude, []provider.Provider{groq, claude}},
		{"groq wins over openrouter", "g", "o", "", noClaude, []provider.Provider{groq}},
		{"openrouter fallback", "", "o", "", withClaude, []provider.Provider{or, claude}},
		{"claude only", "", "", "", withClaude, []provider.Provider{claude}},
		{"MULL_MODEL applies to the API", "g", "", "qwen/qwen3.8-27b", noClaude,
			[]provider.Provider{provider.OpenAI{BaseURL: groqURL, APIKey: "g", Model: "qwen/qwen3.8-27b"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GROQ_API_KEY", tt.groq)
			t.Setenv("OPENROUTER_API_KEY", tt.openrouter)
			t.Setenv("MULL_MODEL", tt.model)
			t.Setenv("PATH", tt.path)
			got, err := backends()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v\nwant %+v", got, tt.want)
			}
		})
	}

	t.Run("nothing configured", func(t *testing.T) {
		t.Setenv("GROQ_API_KEY", "")
		t.Setenv("OPENROUTER_API_KEY", "")
		t.Setenv("PATH", noClaude)
		if _, err := backends(); err == nil {
			t.Fatal("want error, got nil")
		}
	})
}
