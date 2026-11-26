package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEnvLine(t *testing.T) {
	tests := []struct {
		line, key, val string
		ok             bool
	}{
		{"MULL_A=plain", "MULL_A", "plain", true},
		{"export MULL_B=exported", "MULL_B", "exported", true},
		{"export\tMULL_C=tabbed", "MULL_C", "tabbed", true},
		{"MULL_D=value # a note", "MULL_D", "value", true},
		{"MULL_E=\"quoted # kept\" # note", "MULL_E", "quoted # kept", true},
		{"MULL_F='single # kept'", "MULL_F", "single # kept", true},
		{"MULL_G=abc#def", "MULL_G", "abc#def", true},
		{"  MULL_H = spaced  ", "MULL_H", "spaced", true},
		{"# comment", "", "", false},
		{"not a pair", "", "", false},
		{"=no key", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			key, val, ok := parseEnvLine(tt.line)
			if key != tt.key || val != tt.val || ok != tt.ok {
				t.Errorf("parseEnvLine(%q) = %q, %q, %v; want %q, %q, %v", tt.line, key, val, ok, tt.key, tt.val, tt.ok)
			}
		})
	}
}

func TestLoadEnvReportsSetenvError(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("BAD\x00KEY=value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := loadEnv(path)
	if err == nil || !strings.Contains(err.Error(), "setting") {
		t.Fatalf("err = %v, want an error about setting the key", err)
	}
}

func TestLoadEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\n\nMULL_T_A=plain\nMULL_T_B=\"quoted\"\nMULL_T_C=from-file\nnot a pair\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MULL_T_C", "from-env") // real env must win

	if err := loadEnv(path); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"MULL_T_A": "plain", "MULL_T_B": "quoted", "MULL_T_C": "from-env"} {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}

	if err := loadEnv(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Errorf("missing file should be fine, got %v", err)
	}
}

func TestLoadEnvFilesPrecedence(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	if err := os.MkdirAll(filepath.Join(cfg, "mull"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "mull", ".env"), []byte("MULL_T_X=config\nMULL_T_Y=config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	local := t.TempDir()
	if err := os.WriteFile(filepath.Join(local, ".env"), []byte("MULL_T_X=local\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(local)
	// t.Setenv restores the old value after the test; Unsetenv makes it truly unset.
	t.Setenv("MULL_T_X", "")
	os.Unsetenv("MULL_T_X")
	t.Setenv("MULL_T_Y", "")
	os.Unsetenv("MULL_T_Y")

	if err := loadEnvFiles(); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"MULL_T_X": "local", "MULL_T_Y": "config"} {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}
