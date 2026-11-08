<p align="center">
  <img src="assets/mascot.gif" alt="mull mascot: a sleepy grey pebble" width="160">
</p>

<h1 align="center">mull</h1>

<p align="center">
  Bring-your-own-AI helper for the Linux desktop (Wayland/Hyprland first).
</p>

Rewrite selected text, summarize a screen region, ask a quick question — using
Claude, Gemini, OpenRouter, or a local model. You bring the key; mull does the plumbing.

> **Status: early.** `mull rewrite` works today, on Groq with the Claude CLI as fallback.
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
# paste your key from https://console.groq.com/keys
```

| Variable             | Default              | Notes                                                    |
|----------------------|----------------------|----------------------------------------------------------|
| `GROQ_API_KEY`       | —                    | Main backend: fast, free tier of 1,000 requests/day      |
| `OPENROUTER_API_KEY` | —                    | Used instead of Groq when no Groq key is set             |
| `MULL_MODEL`         | `openai/gpt-oss-20b` (Groq), `openrouter/free` (OpenRouter) | Model for the API backend |
| `MULL_PRESET_<NAME>` | —                    | Custom rewrite preset, see [Rewrite modes](#rewrite-modes) |

You need at least one of: a Groq key, an OpenRouter key, or a logged-in
[`claude`](https://docs.anthropic.com/en/docs/claude-code) CLI.

### Backends and fallback

mull tries the API first for speed (Groq ~1s), then falls back to the `claude`
CLI on your Claude subscription (Haiku, ~4s) when the API errors (rate limit, bad
key, outage) or its reply fails the [guardrail](#guardrail). Ctrl-C and the 60s
timeout stop everything. Run with `-v` to see which backend answered.

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

Exit codes: `0` success, `1` runtime error, `2` bad usage, `130` interrupted.

### Rewrite modes

`mull rewrite` on its own only fixes spelling, grammar and punctuation, and keeps
your wording. Add a preset or your own instruction for anything else:

| Command                          | Does                                         |
|----------------------------------|----------------------------------------------|
| `mull rewrite` / `rewrite fix`   | fix typos and grammar, change as little as possible |
| `mull rewrite shorten`           | make it shorter                              |
| `mull rewrite polite`            | make it more polite and friendly             |
| `mull rewrite casual`            | make it more casual                          |
| `mull rewrite translate`         | translate into English                       |
| `mull rewrite make it sound less angry` | any other instruction, quoted or not  |

Every mode keeps the text's language unless the instruction says otherwise
(`mull rewrite translate to Hindi`).

**Custom presets.** Define your own, or override a built-in one, with
`MULL_PRESET_<NAME>` in the environment or a `.env` file. Dashes in the name map to
underscores:

```sh
# ~/.config/mull/.env
MULL_PRESET_FORMAL="Make the text formal and professional."
MULL_PRESET_GIT_MSG="Turn the text into a one-line conventional commit message."
```

```sh
echo "fixed the login bug lol" | mull rewrite git-msg
```

### Guardrail

Small models sometimes answer the text instead of rewriting it. mull sends the
text as marked-up data, then checks the reply: for built-in presets its length
must stay in a sane range (`shorten` must not grow, `fix` stays close to the
original). A reply outside it, or an empty one, goes to the next backend; if
every backend fails the check, mull exits with an error instead of printing a
wrong answer. Free-form instructions and custom presets skip the length check.

## Hotkey (Hyprland)

[`scripts/mull-rewrite`](scripts/mull-rewrite) rewrites the highlighted text
(Wayland primary selection) and puts the result on the clipboard, with a
notification while it works. Needs `wl-clipboard` and `libnotify`.

```sh
go build -o ~/.local/bin/mull ./cmd/mull
ln -s "$PWD/scripts/mull-rewrite" ~/.local/bin/mull-rewrite
ln -s "$PWD/scripts/mull-menu" ~/.local/bin/mull-menu      # optional, needs rofi
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

Highlight text, press the key, paste. Arguments pass through to `mull rewrite`,
so you can bind more keys to presets, e.g. `exec, mull-rewrite shorten`.

**Menu.** [`scripts/mull-menu`](scripts/mull-menu) opens a rofi list of every
preset (custom ones included, from `mull presets`). Pick one, or type any
instruction and press Enter. Escape cancels.

```ini
bind = SUPER, M, exec, mull-menu
```

## Commands

| Command     | What it does                         | Status  |
|-------------|--------------------------------------|---------|
| `rewrite`   | Fix, reshape or translate stdin ([modes](#rewrite-modes)) | works   |
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
| Groq                          | works (default) |
| `claude` CLI (subscription)   | works (fallback) |
| OpenRouter                    | works   |
| Ollama / LM Studio            | planned (same OpenAI-compatible adapter, different base URL) |
| `gemini` CLI                  | planned |

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
- Later: region OCR, keyring key storage, packaging (AUR)

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
