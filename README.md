<p align="center">
  <img src="assets/mascot.gif" alt="mull mascot: a sleepy grey pebble" width="160">
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

mull reads its settings from environment variables, falling back to `.env` files.
The first source that sets a variable wins:

1. Real environment variables
2. `./.env` in the current directory (handy during development)
3. `~/.config/mull/.env` (or `$XDG_CONFIG_HOME/mull/.env`)

```sh
mkdir -p ~/.config/mull
cp .env.example ~/.config/mull/.env
chmod 600 ~/.config/mull/.env
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

Keep your key in `~/.config/mull/.env` and this works from a hotkey in any directory.

Exit codes: `0` success, `1` runtime error, `2` bad usage.

## Hotkey (Hyprland)

[`scripts/mull-rewrite`](scripts/mull-rewrite) rewrites the highlighted text
(Wayland primary selection) and puts the result on the clipboard, with a
notification while it works. Needs `wl-clipboard` and `libnotify`.

```sh
go build -o ~/.local/bin/mull ./cmd/mull
ln -s "$PWD/scripts/mull-rewrite" ~/.local/bin/mull-rewrite
```

Symlink rather than copy, so the script finds the mascot icon in `assets/`.
Then bind a free key (check `hyprctl binds` first):

```ini
# hyprland.conf
bind = SUPER ALT, R, exec, mull-rewrite
```

```lua
-- hyprland.lua
hl.bind("SUPER + ALT + R", hl.dsp.exec_cmd("mull-rewrite"), {
	description = "rewrite selection to clipboard",
})
```

Highlight text, press the key, paste.

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
	Complete(ctx context.Context, instruction, input string) (Reply, error)
}
```

## Roadmap

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
scripts/             Desktop glue (hotkey helpers)
assets/              Mascot images
```
