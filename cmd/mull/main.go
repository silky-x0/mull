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
	"strings"
	"syscall"
	"time"
)

const usage = `usage: echo "text" | mull [-v] rewrite [` + presetNames + ` | instruction...]
       mull presets      list preset names, custom ones included

  no argument   fix spelling and grammar only
  instruction   anything else, e.g. mull rewrite "make it sound less angry"

custom presets: set MULL_PRESET_<NAME>="instruction" (env or .env), then mull rewrite <name>`

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
		if err := rewrite(*verbose, flag.Args()[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "mull:", err)
			if errors.Is(err, errInterrupted) {
				os.Exit(130)
			}
			os.Exit(1)
		}
	case "presets":
		fmt.Println(strings.Join(presetList(), "\n"))
	default:
		flag.Usage()
		os.Exit(2)
	}
}

func rewrite(verbose bool, args []string) error {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	input := strings.TrimSpace(string(raw))
	if input == "" {
		return errors.New("no input text on stdin")
	}

	attempts, err := backends()
	if err != nil {
		return err
	}
	if len(attempts) == 1 {
		attempts = append(attempts, attempts[0]) // no fallback: retry once
	}

	// Set up after reading stdin: once NotifyContext catches SIGINT, Ctrl-C no
	// longer kills the process, and io.ReadAll can't be cancelled.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	msg := rewriteMessage(rewriteInstruction(args), input)
	var lastErr error
	for i, p := range attempts {
		reply, err := p.Complete(ctx, rewriteSystem, msg)
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			return fmt.Errorf("timed out after %s", timeout)
		case errors.Is(err, context.Canceled):
			return errInterrupted
		case err == nil:
			if verbose {
				fmt.Fprintln(os.Stderr, "mull: model:", reply.Model)
			}
			text := unwrapText(reply.Text)
			if err = checkRewrite(args, input, text); err == nil {
				fmt.Println(text)
				return nil
			}
		}
		lastErr = err
		if verbose && i < len(attempts)-1 {
			fmt.Fprintln(os.Stderr, "mull: trying next backend:", err)
		}
	}
	if errors.Is(lastErr, errOffTask) {
		return fmt.Errorf("%w; try again, or pin a stronger model with MULL_MODEL", lastErr)
	}
	return lastErr
}
