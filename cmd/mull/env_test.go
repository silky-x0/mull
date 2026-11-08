package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
