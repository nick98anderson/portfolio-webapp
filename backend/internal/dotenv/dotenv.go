// Package dotenv loads KEY=value lines from a .env file into the process
// environment, without overriding variables that are already set.
package dotenv

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LoadFromRepo finds the nearest .env walking up from the current directory,
// stopping at the repo root (the directory containing .git), and loads it.
// So it works from the repo root, backend/ or anywhere below. A missing file
// is not an error. It returns the path it loaded, or "".
func LoadFromRepo() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return loadFrom(dir)
}

func loadFrom(dir string) (string, error) {
	for {
		path := filepath.Join(dir, ".env")
		if _, err := os.Stat(path); err == nil {
			return path, Load(path)
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return "", nil // reached the repo root without finding one
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// Load reads one .env file. Blank lines and # comments are skipped, an
// optional "export " prefix is allowed, and matching surrounding quotes are
// stripped from values.
func Load(path string) error {
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
		key, val, ok := parseLine(sc.Text())
		if !ok {
			continue
		}
		if _, set := os.LookupEnv(key); !set {
			os.Setenv(key, val)
		}
	}
	return sc.Err()
}

func parseLine(line string) (key, val string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")
	key, val, ok = strings.Cut(line, "=")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false
	}
	val = strings.TrimSpace(val)
	if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[len(val)-1] == val[0] {
		val = val[1 : len(val)-1]
	} else if i := strings.Index(val, " #"); i >= 0 {
		val = strings.TrimSpace(val[:i]) // trailing comment on an unquoted value
	}
	return key, val, true
}
