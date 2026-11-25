package main

import (
	"flag"
	"reflect"
	"testing"
)

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{
		"bg":              "bg",
		"Reef Dive 2026":  "reef-dive-2026",
		"__weird--name!!": "weird-name",
		"bg-portrait":     "bg-portrait",
	} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseInterleaved(t *testing.T) {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	portrait := fs.String("portrait", "", "")
	dry := fs.Bool("dry-run", false, "")
	pos, err := parseInterleaved(fs, []string{"reef.mp4", "--portrait", "p.mp4", "--dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(pos, []string{"reef.mp4"}) || *portrait != "p.mp4" || !*dry {
		t.Fatalf("pos=%v portrait=%q dry=%v", pos, *portrait, *dry)
	}
}

func TestHumanBytes(t *testing.T) {
	for n, want := range map[int64]string{512: "512 B", 2048: "2.0 KB", 6601104: "6.3 MB"} {
		if got := humanBytes(n); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", n, got, want)
		}
	}
}
