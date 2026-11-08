package main

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"strings"
)

// loadEnv reads KEY=VALUE lines from path into the process environment.
// Variables already set in the real environment win over the file.
// A missing file is not an error.
//
// shortcut: reads .env from the current directory only; move to
// ~/.config/mull/ once mull runs from hotkeys (different working dir).
func loadEnv(path string) error {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, val)
		}
	}
	return sc.Err()
}
