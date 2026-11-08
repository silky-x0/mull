package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeClaude puts a `claude` script first on PATH; it records its args and stdin.
func fakeClaude(t *testing.T, script string) (argsFile, stdinFile string) {
	t.Helper()
	dir := t.TempDir()
	argsFile, stdinFile = filepath.Join(dir, "args"), filepath.Join(dir, "stdin")
	body := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\ncat > " + stdinFile + "\n" + script + "\n"
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return argsFile, stdinFile
}

func TestClaudeCLIComplete(t *testing.T) {
	argsFile, stdinFile := fakeClaude(t, `echo '{"result":"Hello.","is_error":false,"modelUsage":{"claude-haiku-5-5":{}}}'`)

	got, err := ClaudeCLI{Model: "haiku"}.Complete(context.Background(), "be brief", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Hello." || got.Model != "claude-haiku-5-5" {
		t.Errorf("got %+v", got)
	}
	args, _ := os.ReadFile(argsFile)
	for _, want := range []string{"-p", "--system-prompt\nbe brief", "--model\nhaiku", "--tools\n\n"} {
		if !strings.Contains(string(args), want) {
			t.Errorf("args missing %q:\n%s", want, args)
		}
	}
	if stdin, _ := os.ReadFile(stdinFile); string(stdin) != "hi" {
		t.Errorf("stdin = %q, want %q", stdin, "hi")
	}
}

func TestClaudeCLIError(t *testing.T) {
	fakeClaude(t, `echo '{"result":"Not logged in · Please run /login","is_error":true}'`)
	_, err := ClaudeCLI{}.Complete(context.Background(), "x", "y")
	if err == nil || !strings.Contains(err.Error(), "Not logged in") {
		t.Fatalf("err = %v, want the CLI's message", err)
	}
}

func TestClaudeCLITimeout(t *testing.T) {
	fakeClaude(t, "sleep 5")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := ClaudeCLI{}.Complete(ctx, "x", "y")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}
