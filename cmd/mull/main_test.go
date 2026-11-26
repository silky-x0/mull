package main

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/silky-x0/mull/internal/provider"
)

type fakeProvider struct {
	text  string
	model string
	err   error
}

func (f fakeProvider) Complete(context.Context, string, string) (provider.Reply, error) {
	return provider.Reply{Text: f.text, Model: f.model}, f.err
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		verbose, help bool
		rest          []string
	}{
		{"flag before command", []string{"-v", "rewrite", "shorten"}, true, false, []string{"rewrite", "shorten"}},
		{"flag after command", []string{"rewrite", "-v"}, true, false, []string{"rewrite"}},
		{"long flag after instruction", []string{"rewrite", "fix", "--verbose"}, true, false, []string{"rewrite", "fix"}},
		{"-v inside instruction is data", []string{"rewrite", "use -v flag"}, false, false, []string{"rewrite", "use -v flag"}},
		{"double dash ends flags", []string{"--", "-v", "rewrite"}, false, false, []string{"-v", "rewrite"}},
		{"help", []string{"-h", "rewrite"}, false, true, []string{"rewrite"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verbose, help, rest := parseArgs(tt.args)
			if verbose != tt.verbose || help != tt.help {
				t.Errorf("parseArgs(%q) = verbose %v, help %v", tt.args, verbose, help)
			}
			if !slices.Equal(rest, tt.rest) {
				t.Errorf("rest = %q, want %q", rest, tt.rest)
			}
		})
	}
}

func TestRunRewriteKeepsAllErrors(t *testing.T) {
	backends := []provider.Provider{
		fakeProvider{err: errors.New("groq: HTTP 429")},
		fakeProvider{err: errors.New("claude: not logged in")},
	}
	var stdout, stderr bytes.Buffer
	err := runRewrite(context.Background(), backends, nil, "teh cat", false, &stdout, &stderr)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	for _, want := range []string{"groq: HTTP 429", "claude: not logged in"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

func TestRunRewriteFallsBack(t *testing.T) {
	backends := []provider.Provider{
		fakeProvider{err: errors.New("groq: HTTP 429")},
		fakeProvider{text: "The cat.", model: "vendor/model"},
	}
	var stdout, stderr bytes.Buffer
	if err := runRewrite(context.Background(), backends, nil, "teh cat", true, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "The cat." {
		t.Errorf("stdout = %q", got)
	}
	if !strings.Contains(stderr.String(), "vendor/model") {
		t.Errorf("stderr = %q, want the model name", stderr.String())
	}
}

func TestRunRewriteOffTaskHint(t *testing.T) {
	tutorial := strings.Repeat("hooks are functions. ", 10)
	backends := []provider.Provider{fakeProvider{text: tutorial}}
	var stdout, stderr bytes.Buffer
	err := runRewrite(context.Background(), backends, []string{"shorten"}, "question?", false, &stdout, &stderr)
	if !errors.Is(err, errOffTask) {
		t.Fatalf("err = %v, want errOffTask", err)
	}
	if !strings.Contains(err.Error(), "pin a stronger model") {
		t.Errorf("error %q is missing the hint", err)
	}
}
