// Command mull is a bring-your-own-AI helper for the Linux desktop.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/silky-x0/mull/internal/provider"
)

const usage = `usage: echo "text" | mull [-v] rewrite`

// openrouter/free routes to whichever free model is currently available.
const defaultModel = "openrouter/free"

// Free models can take a while to answer; this only stops a hung request.
const timeout = 60 * time.Second

var errInterrupted = errors.New("interrupted")

func main() {
	if err := loadEnvFiles(); err != nil {
		fmt.Fprintln(os.Stderr, "mull:", err)
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
			if errors.Is(err, errInterrupted) {
				os.Exit(130)
			}
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

	// Set up after reading stdin: once NotifyContext catches SIGINT, Ctrl-C no
	// longer kills the process, and io.ReadAll can't be cancelled.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	reply, err := p.Complete(ctx,
		"Rewrite the text clearly and correctly. Output only the rewritten text.",
		string(input))
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("timed out after %s", timeout)
	case errors.Is(err, context.Canceled):
		return errInterrupted
	case err != nil:
		return err
	}
	if verbose {
		fmt.Fprintln(os.Stderr, "mull: model:", reply.Model)
	}
	fmt.Println(reply.Text)
	return nil
}
