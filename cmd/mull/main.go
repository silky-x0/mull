// Command mull is a bring-your-own-AI helper for the Linux desktop.
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/silky-x0/mull/internal/provider"
)

const usage = `usage: echo "text" | mull rewrite`

// openrouter/free routes to whichever free model is currently available.
const defaultModel = "openrouter/free"

func main() {
	if err := loadEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "mull: .env:", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "rewrite":
		if err := rewrite(); err != nil {
			fmt.Fprintln(os.Stderr, "mull:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

func rewrite() error {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENROUTER_API_KEY not set (get one at https://openrouter.ai/keys)")
	}
	model := os.Getenv("MULL_MODEL")
	if model == "" {
		model = defaultModel
	}

	var p provider.Provider = provider.OpenRouter{
		BaseURL: "https://openrouter.ai/api/v1",
		APIKey:  key,
		Model:   model,
	}
	out, err := p.Complete(context.Background(),
		"Rewrite the text clearly and correctly. Output only the rewritten text.",
		string(input))
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
