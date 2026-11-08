<p align="center">
  <img src="assets/mascot.png" alt="mull mascot: a sleepy grey pebble" width="160">
</p>

<h1 align="center">mull</h1>

<p align="center">
  Bring-your-own-AI helper for the Linux desktop (Wayland/Hyprland first).
</p>

Rewrite selected text, summarize a screen region, ask a quick question — using
Claude, Gemini, OpenRouter, or a local model. You bring the key; mull does the plumbing.

> **Status: early.** Only `mull rewrite` via OpenRouter works today.
> See [Roadmap](#roadmap) for what's next.

## Install

Requires Go 1.27+.

```sh
git clone https://github.com/silky-x0/mull
cd mull
go build ./cmd/mull
```

## Configure

mull reads an API key from the environment or from a `.env` file in the current directory.
Real environment variables win over `.env`.

```sh
cp .env.example .env
# paste your key from https://openrouter.ai/keys
```

| Variable             | Required | Default           | Notes                                         |
|----------------------|----------|-------------------|-----------------------------------------------|
| `OPENROUTER_API_KEY` | yes      | —                 |                                               |
| `MULL_MODEL`         | no       | `openrouter/free` | Any OpenRouter model id, e.g. `google/gemma-4-31b-it:free` |

`openrouter/free` routes to whichever free model is currently available.

## Usage

```sh
echo "hey whats up cant come tmrw" | ./mull rewrite
```

Add `-v` to see which model answered (printed to stderr, so pipes stay clean):

```sh
echo "hey whats up cant come tmrw" | ./mull -v rewrite
# mull: model: cohere/north-mini-code:free
# Hey, what's up? Can't come tomorrow.
```

mull reads text on stdin and prints the result on stdout, so it composes with
Wayland clipboard tools:

```sh
wl-paste -p | mull rewrite | wl-copy   # rewrite the current selection into the clipboard
```

`.env` is only read from the current directory, so for hotkeys set
`OPENROUTER_API_KEY` in your session environment instead.

Exit codes: `0` success, `1` runtime error, `2` bad usage.

## Commands

| Command     | What it does                         | Status  |
|-------------|--------------------------------------|---------|
| `rewrite`   | Rewrite stdin clearly and correctly  | works   |
| `summarize` | Summarize text or a screen region    | planned |
| `ask`       | Answer a quick question              | planned |

<p>
  <img src="assets/mascot-rewrite.png" alt="rewrite" width="96">
  <img src="assets/mascot-summarize.png" alt="summarize" width="96">
  <img src="assets/mascot-ask.png" alt="ask" width="96">
</p>

## Providers

| Provider                      | Status  |
|-------------------------------|---------|
| OpenRouter                    | works   |
| Ollama / Groq / LM Studio     | planned (same OpenAI-compatible adapter, different base URL) |
| Gemini, Anthropic API keys    | planned |
| `claude -p` / `gemini -p` CLI | planned |

Every backend implements one interface in `internal/provider`:

```go
type Provider interface {
	Complete(ctx context.Context, instruction, input string) (string, error)
}
```

## Roadmap

- Config in `~/.config/mull/` so hotkeys work from any directory
- Hyprland keybinding recipe with `notify-send`
- Timeouts and Ctrl-C handling
- `ask` and `summarize` commands
- Streaming output
- Later: region OCR, `fuzzel` menu, keyring key storage, packaging (AUR)

## Development

```sh
go test ./...
```

```
cmd/mull/            CLI entry point and .env loader
internal/provider/   Provider interface + one file per AI backend
assets/              Mascot images
```
