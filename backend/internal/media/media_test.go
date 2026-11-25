package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"os/exec"
	"path/filepath"
	"testing"
)

// box builds an MP4 box with a 32-bit size header.
func box(typ string, payload int) []byte {
	b := make([]byte, 8+payload)
	binary.BigEndian.PutUint32(b, uint32(8+payload))
	copy(b[4:], typ)
	return b
}

// box64 builds an MP4 box with the 64-bit "largesize" header.
func box64(typ string, payload int) []byte {
	b := make([]byte, 16+payload)
	binary.BigEndian.PutUint32(b, 1)
	copy(b[4:], typ)
	binary.BigEndian.PutUint64(b[8:], uint64(16+payload))
	return b
}

func cat(parts ...[]byte) *bytes.Reader {
	return bytes.NewReader(bytes.Join(parts, nil))
}

func TestIsFaststart(t *testing.T) {
	tests := []struct {
		name    string
		file    *bytes.Reader
		want    bool
		wantErr bool
	}{
		{"moov first", cat(box("ftyp", 16), box("moov", 100), box("mdat", 1000)), true, false},
		{"mdat first", cat(box("ftyp", 16), box("mdat", 1000), box("moov", 100)), false, false},
		{"free box skipped", cat(box("ftyp", 16), box("free", 8), box("moov", 10)), true, false},
		{"64-bit mdat first", cat(box("ftyp", 16), box64("mdat", 50), box("moov", 10)), false, false},
		{"64-bit box skipped", cat(box("ftyp", 16), box64("free", 50), box("moov", 10)), true, false},
		{"not an mp4", cat(box("ftyp", 16), box("free", 8)), false, true},
		{"corrupt size", cat([]byte{0, 0, 0, 4, 'f', 't', 'y', 'p'}), false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isFaststart(tt.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestPipeline runs the real ffmpeg steps on a generated clip. Skipped when
// ffmpeg isn't installed.
func TestPipeline(t *testing.T) {
	if CheckTools() != nil {
		t.Skip("ffmpeg not installed")
	}
	ctx := context.Background()
	dir := t.TempDir()
	src := filepath.Join(dir, "src.mp4")
	// A 2s 640x360 test pattern with a silent audio track, index at the end.
	gen := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "testsrc2=size=640x360:rate=30:duration=2",
		"-f", "lavfi", "-i", "anullsrc", "-shortest", "-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generating test clip: %v: %s", err, out)
	}
	if fast, err := IsFaststart(src); err != nil || fast {
		t.Fatalf("source should not be faststart: fast=%v err=%v", fast, err)
	}

	fast := filepath.Join(dir, "fast.mp4")
	if err := Faststart(ctx, src, fast); err != nil {
		t.Fatal(err)
	}
	if ok, err := IsFaststart(fast); err != nil || !ok {
		t.Fatalf("Faststart output: fast=%v err=%v", ok, err)
	}
	if out, _ := exec.Command("ffprobe", "-v", "error", "-select_streams", "a", "-show_entries", "stream=index", "-of", "csv=p=0", fast).Output(); len(bytes.TrimSpace(out)) != 0 {
		t.Fatal("audio track should be dropped")
	}

	portrait := filepath.Join(dir, "portrait.mp4")
	if err := CropPortrait(ctx, fast, portrait); err != nil {
		t.Fatal(err)
	}
	if info, err := Probe(ctx, portrait); err != nil || info != (Info{Width: 202, Height: 360}) {
		t.Fatalf("portrait = %+v, err %v; want 202x360", info, err)
	}

	poster := filepath.Join(dir, "poster.jpg")
	if err := Poster(ctx, fast, poster, 0); err != nil {
		t.Fatal(err)
	}
	if info, err := Probe(ctx, poster); err != nil || info != (Info{Width: 640, Height: 360}) {
		t.Fatalf("poster = %+v, err %v", info, err)
	}

	blur := filepath.Join(dir, "blur.jpg")
	if err := Blur(ctx, fast, blur, 0); err != nil {
		t.Fatal(err)
	}
	if info, err := Probe(ctx, blur); err != nil || info != (Info{Width: 480, Height: 270}) {
		t.Fatalf("blur = %+v, err %v", info, err)
	}
}
