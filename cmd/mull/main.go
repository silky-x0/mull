// Command mull is a bring-your-own-AI helper for the Linux desktop.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/silky-x0/mull/internal/provider"
)

const usage = `usage: echo "text" | mull [-v] rewrite`

// openrouter/free routes to whichever free model is currently available.
const defaultModel = "openrouter/free"

func main() {
	if err := loadEnv(".env"); err != nil {
		fmt.Fprintln(os.Stderr, "mull: .env:", err)
		os.Exit(1)
	}

	verbose := flag.Bool("v", false, "print which model answered (to stderr)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	switch flag.Arg(0) {
	case "rewrite":
		if err := rewrite(*verbose); err != nil {
			fmt.Fprintln(os.Stderr, "mull:", err)
			os.Exit(1)
		}
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func rewrite(verbose bool) error {
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
	reply, err := p.Complete(context.Background(),
		"Rewrite the text clearly and correctly. Output only the rewritten text.",
		string(input))
	if err != nil {
		return err
	}
	if verbose {
		fmt.Fprintln(os.Stderr, "mull: model:", reply.Model)
	}
	fmt.Println(reply.Text)
	return nil
}
