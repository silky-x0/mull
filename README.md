# mull

Bring-your-own-AI helper for the Linux desktop (Wayland/Hyprland first).
Rewrite selected text, summarize a screen region, ask a quick question — using
Claude, Gemini, OpenRouter, or a local model.

## Status

Early scaffold. Only `rewrite` via OpenRouter works.

```sh
cp .env.example .env   # then paste your key from https://openrouter.ai/keys
go build ./cmd/mull
echo "hey whats up cant come tmrw" | ./mull rewrite
```

## Layout

```
cmd/mull/            CLI entry point
internal/provider/   Provider interface + one file per AI backend
```
