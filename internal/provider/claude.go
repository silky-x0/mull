package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ClaudeCLI runs the logged-in `claude` CLI (Claude Code) in print mode, so it
// uses the user's Claude subscription instead of an API key.
type ClaudeCLI struct {
	Model string // CLI alias or id, e.g. "haiku"; empty uses the CLI default
}

func (c ClaudeCLI) Complete(ctx context.Context, instruction, input string) (Reply, error) {
	// No tools, no saved session, no user settings or hooks: a plain completion.
	// Not --bare: it ignores the subscription login and wants an API key.
	args := []string{"-p", "--tools", "", "--no-session-persistence", "--setting-sources", "",
		"--system-prompt", instruction, "--output-format", "json"}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Stdin = strings.NewReader(input)
	// On cancel only claude itself is killed; its children can keep the output
	// pipe open, so stop waiting for them shortly after.
	cmd.WaitDelay = time.Second
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, runErr := cmd.Output()
	if ctx.Err() != nil {
		return Reply{}, ctx.Err()
	}
	var res struct {
		Result     string                     `json:"result"`
		IsError    bool                       `json:"is_error"`
		ModelUsage map[string]json.RawMessage `json:"modelUsage"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		if runErr != nil {
			return Reply{}, fmt.Errorf("claude: %w: %s", runErr, strings.TrimSpace(stderr.String()))
		}
		return Reply{}, fmt.Errorf("claude: unreadable output: %w", err)
	}
	if res.IsError {
		return Reply{}, fmt.Errorf("claude: %s", res.Result)
	}
	var model string
	for m := range res.ModelUsage {
		model = m
	}
	return Reply{Text: res.Result, Model: model}, nil
}
