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

var usage = `usage: echo "text" | mull [-v] rewrite [` + presetNames + ` | instruction...]
       mull presets      list preset names, custom ones included
       mull ask <question>            answer a question
       mull summarize                 summarize text from stdin
       mull transcribe <audio-file>   speech to text (Groq Whisper, needs GROQ_API_KEY)
       mull version      print the version

  -v, --verbose   print which model answered (to stderr); works before or after the command
  -s, --stream    print tokens as they arrive (OpenAI backends; with Claude the final text is printed)
  no argument     fix spelling and grammar only
  instruction     anything else, e.g. mull rewrite "make it sound less angry"

custom presets: set MULL_PRESET_<NAME>="instruction" (env or .env), then mull rewrite <name>`

// Free models can take a while to answer; this only stops a hung request.
const timeout = 60 * time.Second

var (
	errInterrupted = errors.New("interrupted")
	version        = "dev"
)

func main() {
	if err := loadEnvFiles(); err != nil {
		fmt.Fprintln(os.Stderr, "mull:", err)
		os.Exit(1)
	}

	verbose, stream, help, args := parseArgs(os.Args[1:])
	if help || len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		if help {
			return
		}
		os.Exit(2)
	}

	switch args[0] {
	case "rewrite":
		exitOn(rewrite(verbose, stream, args[1:]))
	case "ask":
		exitOn(ask(verbose, stream, args[1:]))
	case "summarize":
		exitOn(summarize(verbose, stream))
	case "transcribe":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, usage)
			os.Exit(2)
		}
		exitOn(transcribe(args[1]))
	case "presets":
		fmt.Println(strings.Join(presetList(), "\n"))
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

func parseArgs(args []string) (verbose, stream, help bool, rest []string) {
	rest = make([]string, 0, len(args))
	for i, a := range args {
		switch a {
		case "-v", "--verbose":
			verbose = true
		case "-s", "--stream":
			stream = true
		case "-h", "--help":
			help = true
		case "--":
			return verbose, stream, help, append(rest, args[i+1:]...)
		default:
			rest = append(rest, a)
		}
	}
	return verbose, stream, help, rest
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

func readInput(f *os.File) (string, error) {
	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("checking stdin: %w", err)
	}
	if info.Mode()&os.ModeCharDevice != 0 {
		return "", errors.New("no input text on stdin: pipe text in, e.g. echo \"text\" | mull rewrite")
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("reading stdin: %w", err)
	}
	input := strings.TrimSpace(string(raw))
	if input == "" {
		return "", errors.New("no input text on stdin")
	}
	return input, nil
}

func requestBackends() ([]provider.Provider, error) {
	attempts, err := backends()
	if err != nil {
		return nil, err
	}
	if len(attempts) == 1 {
		attempts = append(attempts, attempts[0])
	}
	return attempts, nil
}

func rewrite(verbose, stream bool, args []string) error {
	input, err := readInput(os.Stdin)
	if err != nil {
		return err
	}
	attempts, err := requestBackends()
	if err != nil {
		return err
	}
	ctx, cancel := requestContext()
	defer cancel()
	return runRewrite(ctx, attempts, args, input, verbose, stream, os.Stdout, os.Stderr)
}

func runRewrite(ctx context.Context, attempts []provider.Provider, args []string, input string, verbose, stream bool, stdout, stderr io.Writer) error {
	msg := rewriteMessage(rewriteInstruction(args), input)
	err := complete(ctx, attempts, rewriteSystem, msg, verbose, stream, func(text string) error {
		return checkRewrite(args, input, text)
	}, stdout, stderr)
	if errors.Is(err, errOffTask) {
		return fmt.Errorf("%w; try again, or pin a stronger model with MULL_MODEL", err)
	}
	return err
}

func complete(ctx context.Context, attempts []provider.Provider, system, message string, verbose, stream bool, check func(string) error, stdout, stderr io.Writer) error {
	errs := make([]error, 0, len(attempts))
	for i, p := range attempts {
		streamed := false
		var reply provider.Reply
		var err error
		if stream {
			if sp, ok := p.(provider.Streamer); ok {
				reply, err = sp.Stream(ctx, system, message, func(delta string) { fmt.Fprint(stdout, delta) })
				streamed = err == nil
			} else {
				reply, err = p.Complete(ctx, system, message)
			}
		} else {
			reply, err = p.Complete(ctx, system, message)
		}
		if cerr := contextErr(err); cerr != nil {
			return cerr
		}
		if err == nil {
			if verbose {
				fmt.Fprintln(stderr, "mull: model:", reply.Model)
			}
			text := unwrapText(reply.Text)
			if err = check(text); err == nil {
				if streamed {
					fmt.Fprintln(stdout)
				} else {
					fmt.Fprintln(stdout, text)
				}
				return nil
			}
			if streamed {
				return fmt.Errorf("streamed reply failed check: %w", err)
			}
		}
		errs = append(errs, fmt.Errorf("attempt %d: %w", i+1, err))
		if verbose && i < len(attempts)-1 {
			fmt.Fprintln(stderr, "mull: trying next backend:", err)
		}
	}
	return errors.Join(errs...)
}

func checkNonEmpty(text string) error {
	if text == "" {
		return errors.New("model returned empty text")
	}
	return nil
}
