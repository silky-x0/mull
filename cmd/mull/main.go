// Command mull is a bring-your-own-AI helper for the Linux desktop.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/silky-x0/mull/internal/provider"
)

const usage = `usage: echo "text" | mull [-v] rewrite [` + presetNames + ` | instruction...]
       mull presets      list preset names, custom ones included
       mull transcribe <audio-file>   speech to text (Groq Whisper, needs GROQ_API_KEY)

  -v, --verbose   print which model answered (to stderr); works before or after the command
  no argument     fix spelling and grammar only
  instruction     anything else, e.g. mull rewrite "make it sound less angry"

custom presets: set MULL_PRESET_<NAME>="instruction" (env or .env), then mull rewrite <name>`

// Free models can take a while to answer; this only stops a hung request.
const timeout = 60 * time.Second

var errInterrupted = errors.New("interrupted")

func main() {
	if err := loadEnvFiles(); err != nil {
		fmt.Fprintln(os.Stderr, "mull:", err)
		os.Exit(1)
	}

	verbose, help, args := parseArgs(os.Args[1:])
	if help || len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		if help {
			return
		}
		os.Exit(2)
	}

	switch args[0] {
	case "rewrite":
		exitOn(rewrite(verbose, args[1:]))
	case "transcribe":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, usage)
			os.Exit(2)
		}
		exitOn(transcribe(args[1]))
	case "presets":
		fmt.Println(strings.Join(presetList(), "\n"))
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

func parseArgs(args []string) (verbose, help bool, rest []string) {
	rest = make([]string, 0, len(args))
	for i, a := range args {
		switch a {
		case "-v", "--verbose":
			verbose = true
		case "-h", "--help":
			help = true
		case "--":
			return verbose, help, append(rest, args[i+1:]...)
		default:
			rest = append(rest, a)
		}
	}
	return verbose, help, rest
}

func exitOn(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "mull:", err)
	if errors.Is(err, errInterrupted) {
		os.Exit(130)
	}
	os.Exit(1)
}

// requestContext stops on Ctrl-C/SIGTERM or after timeout. Create it after
// reading stdin: once NotifyContext catches SIGINT, Ctrl-C no longer kills the
// process, and io.ReadAll can't be cancelled.
func requestContext() (context.Context, func()) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	return ctx, func() { cancel(); stop() }
}

// contextErr turns a cancelled request into a user-facing error, or nil.
func contextErr(err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("timed out after %s", timeout)
	case errors.Is(err, context.Canceled):
		return errInterrupted
	}
	return nil
}

func rewrite(verbose bool, args []string) error {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
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

	ctx, cancel := requestContext()
	defer cancel()
	return runRewrite(ctx, attempts, args, input, verbose, os.Stdout, os.Stderr)
}

func runRewrite(ctx context.Context, attempts []provider.Provider, args []string, input string, verbose bool, stdout, stderr io.Writer) error {
	msg := rewriteMessage(rewriteInstruction(args), input)
	errs := make([]error, 0, len(attempts))
	for i, p := range attempts {
		reply, err := p.Complete(ctx, rewriteSystem, msg)
		if cerr := contextErr(err); cerr != nil {
			return cerr
		}
		if err == nil {
			if verbose {
				fmt.Fprintln(stderr, "mull: model:", reply.Model)
			}
			text := unwrapText(reply.Text)
			if err = checkRewrite(args, input, text); err == nil {
				fmt.Fprintln(stdout, text)
				return nil
			}
		}
		errs = append(errs, fmt.Errorf("attempt %d: %w", i+1, err))
		if verbose && i < len(attempts)-1 {
			fmt.Fprintln(stderr, "mull: trying next backend:", err)
		}
	}
	allErr := errors.Join(errs...)
	if errors.Is(allErr, errOffTask) {
		return fmt.Errorf("%w; try again, or pin a stronger model with MULL_MODEL", allErr)
	}
	return allErr
}
