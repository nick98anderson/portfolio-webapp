package manifest

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func video(prefix string, uploaded time.Time) *Video {
	return &Video{
		Landscape: Cut{Src: prefix + ".mp4", Poster: prefix + "-poster.jpg"},
		Portrait:  Cut{Src: prefix + "-portrait.mp4", Poster: prefix + "-portrait-poster.jpg"},
		Blur:      prefix + "-blur.jpg",
		Uploaded:  uploaded,
	}
}

func TestPutUseRemove(t *testing.T) {
	t0 := time.Date(2026, 11, 24, 0, 0, 0, 0, time.UTC)
	m := New()

	if m.Put("bg", video("video/bg", t0), t0) {
		t.Fatal("first put reported a replacement")
	}
	if !m.Put("bg", video("video/bg2", t0), t0) {
		t.Fatal("second put of the same name did not report a replacement")
	}
	m.Put("reef", video("video/reef", t0.Add(time.Hour)), t0.Add(time.Hour))
	if m.Current != "reef" {
		t.Fatalf("current = %q, want reef", m.Current)
	}

	if err := m.Remove("reef", t0); err == nil {
		t.Fatal("removing the live video should fail")
	}
	if err := m.Use("nope", t0); err == nil {
		t.Fatal("using a missing video should fail")
	}
	if err := m.Use("bg", t0); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove("reef", t0); err != nil {
		t.Fatal(err)
	}
	if got := m.Names(); !reflect.DeepEqual(got, []string{"bg"}) {
		t.Fatalf("names = %v", got)
	}
}

func TestReferenced(t *testing.T) {
	t0 := time.Now()
	m := New()
	m.Put("bg", video("video/bg", t0), t0)
	refs := m.Referenced()
	for _, k := range []string{Key, "video/bg.mp4", "video/bg-portrait-poster.jpg", "video/bg-blur.jpg"} {
		if !refs[k] {
			t.Errorf("%s not referenced", k)
		}
	}
	if refs["video/other.mp4"] {
		t.Error("unexpected reference")
	}
}

func TestNamesNewestFirst(t *testing.T) {
	t0 := time.Now()
	m := New()
	m.Put("old", video("a", t0), t0)
	m.Put("new", video("b", t0.Add(time.Minute)), t0)
	if got := m.Names(); !reflect.DeepEqual(got, []string{"new", "old"}) {
		t.Fatalf("names = %v", got)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	t0 := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	m := New()
	m.Put("bg", video("video/bg", t0), t0)
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var back Manifest
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m, &back) {
		t.Fatalf("round trip changed the manifest:\n%s", b)
	}
}
