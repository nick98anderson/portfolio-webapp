// Package media prepares background videos for streaming with ffmpeg:
// faststart MP4s, poster frames and a blurred still.
package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// CheckTools reports a clear error when ffmpeg or ffprobe is missing.
func CheckTools() error {
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s not found on PATH (macOS: brew install ffmpeg)", tool)
		}
	}
	return nil
}

// IsFaststart reports whether an MP4's index (moov box) comes before its
// media data (mdat box). Browsers can then start playback after the first
// few kilobytes instead of fetching the end of the file first.
func IsFaststart(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	return isFaststart(f)
}

func isFaststart(r io.ReadSeeker) (bool, error) {
	var hdr [16]byte
	for {
		if _, err := io.ReadFull(r, hdr[:8]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return false, errors.New("no moov or mdat box found: not an MP4?")
			}
			return false, err
		}
		size := int64(binary.BigEndian.Uint32(hdr[:4]))
		typ := string(hdr[4:8])
		headerLen := int64(8)
		switch size {
		case 1: // 64-bit size follows the type
			if _, err := io.ReadFull(r, hdr[8:16]); err != nil {
				return false, err
			}
			size = int64(binary.BigEndian.Uint64(hdr[8:16]))
			headerLen = 16
		case 0: // box runs to the end of the file
			return typ == "moov", nil
		}
		switch typ {
		case "moov":
			return true, nil
		case "mdat":
			return false, nil
		}
		if size < headerLen {
			return false, fmt.Errorf("corrupt %q box (size %d)", typ, size)
		}
		if _, err := r.Seek(size-headerLen, io.SeekCurrent); err != nil {
			return false, err
		}
	}
}

// Info is the size of a video's first video stream.
type Info struct {
	Width, Height int
}

// Probe reads the dimensions of a video or image.
func Probe(ctx context.Context, path string) (Info, error) {
	out, err := run(ctx, "ffprobe", "-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=p=0:s=x", path)
	if err != nil {
		return Info{}, err
	}
	w, h, ok := strings.Cut(strings.TrimSpace(out), "x")
	if !ok {
		return Info{}, fmt.Errorf("ffprobe %s: unexpected output %q", path, out)
	}
	var info Info
	if info.Width, err = strconv.Atoi(w); err != nil {
		return Info{}, err
	}
	if info.Height, err = strconv.Atoi(h); err != nil {
		return Info{}, err
	}
	return info, nil
}

// Faststart rewrites an MP4 with its index at the front and any audio track
// dropped (the background is always muted). Streams are copied, not
// re-encoded, so quality is untouched and it takes about a second.
func Faststart(ctx context.Context, in, out string) error {
	_, err := run(ctx, "ffmpeg", "-v", "error", "-y", "-i", in,
		"-map", "0:v:0", "-c", "copy", "-an", "-movflags", "+faststart", out)
	return err
}

// CropPortrait cuts a 9:16 slice from the middle of a landscape video, for
// phones, when no hand-made portrait cut is supplied. This re-encodes.
func CropPortrait(ctx context.Context, in, out string) error {
	_, err := run(ctx, "ffmpeg", "-v", "error", "-y", "-i", in,
		"-vf", "crop=trunc(ih*9/16/2)*2:ih",
		"-c:v", "libx264", "-preset", "slow", "-crf", "23", "-pix_fmt", "yuv420p",
		"-an", "-movflags", "+faststart", out)
	return err
}

// Poster saves one frame at the given time (seconds) as a JPEG. The default
// of 0 matches the video's first frame, so the swap to playback is seamless.
func Poster(ctx context.Context, in, out string, at float64) error {
	_, err := run(ctx, "ffmpeg", "-v", "error", "-y", "-ss", formatSeconds(at), "-i", in,
		"-frames:v", "1", "-q:v", "3", out)
	return err
}

// Blur saves a small, heavily blurred frame: the calm backdrop behind the
// reading pages. It's scaled up and covered by gradients there, so 480px
// wide is plenty and keeps it to a couple of kilobytes.
func Blur(ctx context.Context, in, out string, at float64) error {
	_, err := run(ctx, "ffmpeg", "-v", "error", "-y", "-ss", formatSeconds(at), "-i", in,
		"-frames:v", "1", "-vf", "scale=480:-2,gblur=sigma=30", "-q:v", "4", out)
	return err
}

func formatSeconds(s float64) string {
	return strconv.FormatFloat(s, 'f', -1, 64)
}

func run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
