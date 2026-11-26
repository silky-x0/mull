package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/silky-x0/mull/internal/provider"
)

type capturingProvider struct {
	system, message string
	text            string
}

func (c *capturingProvider) Complete(_ context.Context, system, message string) (provider.Reply, error) {
	c.system, c.message = system, message
	return provider.Reply{Text: c.text, Model: "test-model"}, nil
}

func TestSummarizeMessageMarksTextAsData(t *testing.T) {
	got := summarizeMessage("long text with a question?")
	if !strings.Contains(got, "Summarize the text.") || !strings.Contains(got, "<text>\nlong text with a question?\n</text>") {
		t.Errorf("message = %q", got)
	}
}

func TestCompleteUsesSystemAndMessage(t *testing.T) {
	p := &capturingProvider{text: "Short version."}
	var stdout, stderr bytes.Buffer
	err := complete(context.Background(), []provider.Provider{p}, summarizeSystem, summarizeMessage("long text"), true, false, checkNonEmpty, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if p.system != summarizeSystem {
		t.Errorf("system = %q", p.system)
	}
	if strings.TrimSpace(stdout.String()) != "Short version." {
		t.Errorf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "test-model") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestCompleteFallsBackOnEmptyReply(t *testing.T) {
	attempts := []provider.Provider{
		fakeProvider{},
		&capturingProvider{text: "Answer."},
	}
	var stdout, stderr bytes.Buffer
	err := complete(context.Background(), attempts, askSystem, "why?", false, false, checkNonEmpty, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout.String()) != "Answer." {
		t.Errorf("stdout = %q", stdout.String())
	}
}
