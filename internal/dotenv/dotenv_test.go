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
