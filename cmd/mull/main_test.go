package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
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

type streamingProvider struct {
	deltas []string
}

func (s *streamingProvider) Complete(context.Context, string, string) (provider.Reply, error) {
	return provider.Reply{}, errors.New("unexpected Complete call")
}

func (s *streamingProvider) Stream(_ context.Context, _, _ string, onDelta func(string)) (provider.Reply, error) {
	for _, d := range s.deltas {
		onDelta(d)
	}
	return provider.Reply{Text: strings.Join(s.deltas, ""), Model: "stream-model"}, nil
}

func TestCompleteStreamsDeltas(t *testing.T) {
	p := &streamingProvider{deltas: []string{"Hel", "lo."}}
	var stdout, stderr bytes.Buffer
	err := complete(context.Background(), []provider.Provider{p}, "sys", "msg", true, true, checkNonEmpty, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "Hello.\n" {
		t.Errorf("stdout = %q, want %q", stdout.String(), "Hello.\n")
	}
	if !strings.Contains(stderr.String(), "stream-model") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestCompleteStreamFallsBackForNonStreamer(t *testing.T) {
	p := fakeProvider{text: "Whole answer.", model: "plain-model"}
	var stdout, stderr bytes.Buffer
	err := complete(context.Background(), []provider.Provider{p}, "sys", "msg", false, true, checkNonEmpty, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "Whole answer.\n" {
		t.Errorf("stdout = %q", stdout.String())
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name                  string
		args                  []string
		verbose, stream, help bool
		rest                  []string
	}{
		{"flag before command", []string{"-v", "rewrite", "shorten"}, true, false, false, []string{"rewrite", "shorten"}},
		{"flag after command", []string{"rewrite", "-v"}, true, false, false, []string{"rewrite"}},
		{"long flag after instruction", []string{"rewrite", "fix", "--verbose"}, true, false, false, []string{"rewrite", "fix"}},
		{"stream flag", []string{"ask", "-s", "why?"}, false, true, false, []string{"ask", "why?"}},
		{"-v inside instruction is data", []string{"rewrite", "use -v flag"}, false, false, false, []string{"rewrite", "use -v flag"}},
		{"double dash ends flags", []string{"--", "-v", "rewrite"}, false, false, false, []string{"-v", "rewrite"}},
		{"help", []string{"-h", "rewrite"}, false, false, true, []string{"rewrite"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verbose, stream, help, rest := parseArgs(tt.args)
			if verbose != tt.verbose || stream != tt.stream || help != tt.help {
				t.Errorf("parseArgs(%q) = verbose %v, stream %v, help %v", tt.args, verbose, stream, help)
			}
			if !slices.Equal(rest, tt.rest) {
				t.Errorf("rest = %q, want %q", rest, tt.rest)
			}
		})
	}
}

func TestReadInput(t *testing.T) {
	dir := t.TempDir()
	withText := filepath.Join(dir, "text")
	if err := os.WriteFile(withText, []byte("  teh cat  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(withText)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readInput(f)
	f.Close()
	if err != nil || got != "teh cat" {
		t.Fatalf("readInput = %q, %v; want %q", got, err, "teh cat")
	}

	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	f, err = os.Open(empty)
	if err != nil {
		t.Fatal(err)
	}
	_, err = readInput(f)
	f.Close()
	if err == nil {
		t.Fatal("want an error for empty input")
	}

	f, err = os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	_, err = readInput(f)
	f.Close()
	if err == nil || !strings.Contains(err.Error(), "pipe text in") {
		t.Fatalf("err = %v, want a hint about piping text", err)
	}
}

func TestRunRewriteKeepsAllErrors(t *testing.T) {
	backends := []provider.Provider{
		fakeProvider{err: errors.New("groq: HTTP 429")},
		fakeProvider{err: errors.New("claude: not logged in")},
	}
	var stdout, stderr bytes.Buffer
	err := runRewrite(context.Background(), backends, nil, "teh cat", false, false, &stdout, &stderr)
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
	if err := runRewrite(context.Background(), backends, nil, "teh cat", true, false, &stdout, &stderr); err != nil {
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
	err := runRewrite(context.Background(), backends, []string{"shorten"}, "question?", false, false, &stdout, &stderr)
	if !errors.Is(err, errOffTask) {
		t.Fatalf("err = %v, want errOffTask", err)
	}
	if !strings.Contains(err.Error(), "pin a stronger model") {
		t.Errorf("error %q is missing the hint", err)
	}
}
