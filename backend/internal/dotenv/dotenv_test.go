package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		line, key, val string
		ok             bool
	}{
		{"A=1", "A", "1", true},
		{"  B = two  ", "B", "two", true},
		{"export C=3", "C", "3", true},
		{`D="quoted # not a comment"`, "D", "quoted # not a comment", true},
		{"E='single'", "E", "single", true},
		{"F=value # trailing comment", "F", "value", true},
		{"G=", "G", "", true},
		{"# comment", "", "", false},
		{"", "", "", false},
		{"no-equals", "", "", false},
		{"=novalue", "", "", false},
	}
	for _, tt := range tests {
		key, val, ok := parseLine(tt.line)
		if key != tt.key || val != tt.val || ok != tt.ok {
			t.Errorf("parseLine(%q) = %q, %q, %v; want %q, %q, %v", tt.line, key, val, ok, tt.key, tt.val, tt.ok)
		}
	}
}

func TestLoadFromWalksUpToRepoRoot(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "backend", "cmd")
	for _, d := range []string{filepath.Join(root, ".git"), sub} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte("DOTENV_TEST_WALK=found\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("DOTENV_TEST_WALK")
	t.Cleanup(func() { os.Unsetenv("DOTENV_TEST_WALK") })

	got, err := loadFrom(sub)
	if err != nil {
		t.Fatal(err)
	}
	if got != envPath || os.Getenv("DOTENV_TEST_WALK") != "found" {
		t.Fatalf("loaded %q, DOTENV_TEST_WALK=%q", got, os.Getenv("DOTENV_TEST_WALK"))
	}

	// No .env anywhere up to the repo root: nothing loaded, and the search
	// doesn't escape above the root.
	os.Remove(envPath)
	if got, err := loadFrom(sub); err != nil || got != "" {
		t.Fatalf("got %q, %v; want nothing", got, err)
	}
}

func TestLoadDoesNotOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("DOTENV_TEST_NEW=from-file\nDOTENV_TEST_SET=from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOTENV_TEST_SET", "from-env")
	os.Unsetenv("DOTENV_TEST_NEW")
	t.Cleanup(func() { os.Unsetenv("DOTENV_TEST_NEW") })

	if err := Load(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("DOTENV_TEST_NEW"); got != "from-file" {
		t.Errorf("DOTENV_TEST_NEW = %q", got)
	}
	if got := os.Getenv("DOTENV_TEST_SET"); got != "from-env" {
		t.Errorf("DOTENV_TEST_SET = %q, should keep the existing value", got)
	}
}
