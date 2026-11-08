package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// loadEnvFiles loads ./.env, then $XDG_CONFIG_HOME/mull/.env
// (~/.config/mull/.env). The first file to set a variable wins, so a local
// .env overrides the user config during development.
func loadEnvFiles() error {
	files := []string{".env"}
	if dir, err := os.UserConfigDir(); err == nil {
		files = append(files, filepath.Join(dir, "mull", ".env"))
	}
	for _, f := range files {
		if err := loadEnv(f); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
	}
	return nil
}

// loadEnv reads KEY=VALUE lines from path into the process environment.
// Variables already set in the real environment win over the file.
// A missing file is not an error.
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
