package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/silky-x0/mull/internal/provider"
)

// Turbo is the faster Whisper; same free quota as large-v3 (8h of audio/day).
const whisperModel = "whisper-large-v3-turbo"

func transcribe(path string) error {
	key := os.Getenv("GROQ_API_KEY")
	if key == "" {
		return errors.New("transcribe needs GROQ_API_KEY (https://console.groq.com/keys)")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	ctx, cancel := requestContext()
	defer cancel()

	stt := provider.OpenAI{BaseURL: groqURL, APIKey: key, Model: whisperModel}
	text, err := stt.Transcribe(ctx, filepath.Base(path), f)
	if cerr := contextErr(err); cerr != nil {
		return cerr
	}
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("no speech detected")
	}
	fmt.Println(text)
	return nil
}
