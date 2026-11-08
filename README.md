<p align="center">
  <img src="assets/mascot.gif" alt="mull mascot: a sleepy grey pebble" width="160">
</p>

<h1 align="center">mull</h1>

<p align="center">
  Bring-your-own-AI helper for the Linux desktop (Wayland/Hyprland first).
</p>

Highlight text and press a key to fix, shorten or reword it. Hold a key and
speak to type by voice. You bring the AI key (or your Claude subscription);
mull does the plumbing.

> **Status: early.** Working today: rewrite (hotkey + preset menu) and
> push-to-talk dictation, on Groq with the Claude CLI as fallback.
> Planned: ask, summarize, screen understanding. See [Roadmap](#roadmap).

## Quick start

Tested on Arch + Hyprland. Other Wayland compositors work too, see
[Other desktops](#other-desktops).

**1. Install dependencies**

| Needed for            | Arch                                | Debian / Ubuntu                          | Fedora                                    |
|-----------------------|-------------------------------------|------------------------------------------|-------------------------------------------|
| building mull         | `go` (1.27+)                        | `golang`                                 | `golang`                                  |
| rewrite hotkey        | `wl-clipboard libnotify`            | `wl-clipboard libnotify-bin`             | `wl-clipboard libnotify`                  |
| preset menu           | `rofi`                              | `rofi`                                   | `rofi` (or `rofi-wayland` on older releases) |
| dictation             | `pipewire wtype`                    | `pipewire-bin wtype`                     | `pipewire-utils wtype`                    |

You also need a notification daemon (mako, dunst, swaync…), which most
desktop setups already run.

**2. Build and install** (into `~/.local/bin`, which must be on your `PATH`)

```sh
git clone https://github.com/silky-x0/mull
cd mull
go build -o ~/.local/bin/mull ./cmd/mull
ln -s "$PWD/scripts/mull-rewrite" ~/.local/bin/
ln -s "$PWD/scripts/mull-menu"    ~/.local/bin/
ln -s "$PWD/scripts/mull-dictate" ~/.local/bin/
```

Symlink the scripts rather than copying them: they find the mascot icons in the
repo through the link, and `git pull` updates them. Don't move the repo
afterwards. After a `git pull`, re-run the `go build` line.

**3. Add your key**

Get a free key at [console.groq.com/keys](https://console.groq.com/keys), then:

```sh
mkdir -p ~/.config/mull
cp .env.example ~/.config/mull/.env
chmod 600 ~/.config/mull/.env
$EDITOR ~/.config/mull/.env        # paste it after GROQ_API_KEY=
```

Optional: install and log in to the [`claude`](https://docs.anthropic.com/en/docs/claude-code)
CLI, and mull falls back to your Claude subscription when Groq fails.

**4. Check it works**

```sh
echo "hey whats up cant come tmrw" | mull -v rewrite
# mull: model: openai/gpt-oss-20b
# Hey, what's up? Can't come tomorrow.
```

**5. Bind keys** in your Hyprland config, see [Hotkeys](#hotkeys-hyprland). The
defaults used below: `SUPER+ALT+R` fix selection, `SUPER+M` preset menu, hold
`F9` to dictate. Pick free keys: `hyprctl binds` lists the ones in use.

## Configure

mull reads its settings from environment variables, falling back to `.env` files.
The first source that sets a variable wins:

1. Real environment variables
2. `./.env` in the current directory (handy during development)
3. `~/.config/mull/.env` (or `$XDG_CONFIG_HOME/mull/.env`)

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
echo "hey whats up cant come tmrw" | mull rewrite
mull transcribe recording.wav
```

Add `-v` to see which model answered (printed to stderr, so pipes stay clean).

mull reads text on stdin and prints the result on stdout, so it composes with
Wayland clipboard tools:

```sh
wl-paste -p | mull rewrite | wl-copy   # rewrite the current selection into the clipboard
```

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

## Hotkeys (Hyprland)

All three scripts are installed in [Quick start](#quick-start) step 2.

**Rewrite.** [`scripts/mull-rewrite`](scripts/mull-rewrite) rewrites the
highlighted text (Wayland primary selection, no Ctrl+C needed) and puts the
result on the clipboard, with a notification while it works.

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

**Dictation (push-to-talk).** [`scripts/mull-dictate`](scripts/mull-dictate)
records the default mic while a key is held, then transcribes with Groq Whisper,
types the text into the focused window (`wtype`) and copies it to the clipboard.
Bind the same key twice, once for press and once for release:

```ini
# hyprland.conf
bind  = , F9, exec, mull-dictate start
bindr = , F9, exec, mull-dictate stop
```

```lua
-- hyprland.lua
hl.bind("F9", hl.dsp.exec_cmd("mull-dictate start"), { description = "dictation: hold to record" })
hl.bind("F9", hl.dsp.exec_cmd("mull-dictate stop"), { release = true, description = "dictation: release to type" })
```

Taps under ~0.3 s are ignored; recordings stop on their own after 10 minutes.
Needs `GROQ_API_KEY` (free tier: 8 hours of audio a day). Pause music and videos
while dictating, the mic picks them up too.

### Other desktops

The scripts only need a Wayland session; only the key binding syntax differs.

- **Sway / river / other wlroots compositors:** everything works. In Sway:
  `bindsym $mod+Alt+r exec mull-rewrite`, and for push-to-talk
  `bindsym F9 exec mull-dictate start` plus `bindsym --release F9 exec mull-dictate stop`.
- **KDE Plasma:** bind the commands in System Settings → Shortcuts. Dictation
  needs a press/release binding, which Plasma's shortcut editor can't do.
- **GNOME:** not really supported. Mutter lacks the virtual keyboard protocol
  `wtype` needs (dictation can't type, only copy) and the clipboard protocol
  `wl-clipboard` prefers, so reading the selection is unreliable.
- **X11:** not supported.

### Troubleshooting

| Symptom | Fix |
|---|---|
| Hotkey does nothing | Run the script in a terminal to see the error, e.g. `mull-rewrite`. Check `~/.local/bin` is on the compositor's `PATH`, or use the full path in the bind. |
| `GROQ_API_KEY not set` from a hotkey but fine in the repo | The hotkey can't see the repo's `.env`. Put the key in `~/.config/mull/.env`. |
| `HTTP 429` / rate limit | Free-tier quota used up. mull falls back to the `claude` CLI if it's installed. |
| Notifications show no icon | The script was copied instead of symlinked. Re-create the link (step 2). |
| `Nothing selected` | Highlight the text again; some apps clear the selection when focus changes. |

### Privacy

The text you rewrite and the audio you dictate are sent to Groq (or OpenRouter),
and on fallback to Anthropic through the `claude` CLI. Read their data policies
before using mull on anything confidential. Your keys stay in
`~/.config/mull/.env` (mode 600) and never leave your machine except to those APIs.

## Commands

| Command     | What it does                         | Status  |
|-------------|--------------------------------------|---------|
| `rewrite`   | Fix, reshape or translate stdin ([modes](#rewrite-modes)) | works   |
| `transcribe`| Speech to text from an audio file (Groq Whisper) | works   |
| `summarize` | Summarize text or a screen region    | planned |
| `ask`       | Answer a quick question              | planned |

<p>
  <img src="assets/mascot-rewrite.png" alt="rewrite" width="96">
  <img src="assets/mascot-summarize.png" alt="summarize" width="96">
  <img src="assets/mascot-ask.png" alt="ask" width="96">
  <img src="assets/mascot-dictate.png" alt="dictate" width="96">
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

- `ask` and `summarize` commands
- Screen understanding (`mull look`) with a vision model
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
