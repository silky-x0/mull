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
		key, val, ok := parseEnvLine(sc.Text())
		if !ok {
			continue
		}
		if _, set := os.LookupEnv(key); !set {
			if err := os.Setenv(key, val); err != nil {
				return fmt.Errorf("setting %s: %w", key, err)
			}
		}
	}
	return sc.Err()
}

func parseEnvLine(line string) (key, val string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	if rest, found := strings.CutPrefix(line, "export"); found && len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
		line = strings.TrimSpace(rest)
	}
	key, raw, ok := strings.Cut(line, "=")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false
	}
	return key, parseEnvValue(raw), true
}

func parseEnvValue(raw string) string {
	v := strings.TrimSpace(raw)
	var quote byte
	for i := 0; i < len(v); i++ {
		switch {
		case quote != 0:
			if v[i] == quote {
				quote = 0
			}
		case v[i] == '"' || v[i] == '\'':
			quote = v[i]
		case v[i] == '#' && i > 0 && (v[i-1] == ' ' || v[i-1] == '\t'):
			v = v[:i]
		}
	}
	v = strings.TrimSpace(v)
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		v = v[1 : len(v)-1]
	}
	return v
}
