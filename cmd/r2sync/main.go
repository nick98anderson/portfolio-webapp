// Command r2sync prepares background videos and publishes them to the R2
// media bucket. Run it locally whenever the site's background should change:
//
//	go run ./cmd/r2sync push ~/Videos/reef.mp4 --portrait ~/Videos/reef-portrait.mp4
//
// Credentials come from R2_* variables, read from .env at the repo root.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nick98anderson/portfolio-webapp/internal/dotenv"
	"github.com/nick98anderson/portfolio-webapp/internal/manifest"
	"github.com/nick98anderson/portfolio-webapp/internal/media"
	"github.com/nick98anderson/portfolio-webapp/internal/r2"
)

// mediaPrefix is where uploads live in the bucket; prune only looks here.
const mediaPrefix = "video/"

const usage = `r2sync: publish background videos to the R2 media bucket

Usage:
  r2sync push <video.mp4> [--portrait <video.mp4>] [--blur <still.jpg>] [--name <name>]
              [--poster-at <sec>] [--dry-run]
      Prepare a video (faststart, posters, blurred still), upload it and make it live.
      Without --portrait, a 9:16 slice is cropped from the middle of the video.
      Without --blur, the blurred still for the resume page is made from the poster frame.
  r2sync list
      Show every uploaded video; * marks the live one.
  r2sync use <name>
      Make an earlier upload live again (rollback).
  r2sync remove <name> [--yes]
      Delete an upload that isn't live. Shows what would go without --yes.
  r2sync prune [--yes]
      Delete files under video/ that no upload points at. Shows them without --yes.
  r2sync cors [--origin <url>]...
      Let the site's origins read the bucket from the browser (needed once).

Environment (from .env at the repo root):
  R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET, R2_PUBLIC_URL
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "r2sync:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprint(out, usage)
		return nil
	}
	if _, err := dotenv.LoadFromRepo(); err != nil {
		return fmt.Errorf("loading .env: %w", err)
	}
	cfg, err := r2.ConfigFromEnv()
	if err != nil {
		return err
	}
	b := r2.New(cfg)

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "push":
		return push(ctx, b, rest, out)
	case "list", "ls":
		return list(ctx, b, out)
	case "use":
		return use(ctx, b, rest, out)
	case "remove", "rm":
		return remove(ctx, b, rest, out)
	case "prune":
		return prune(ctx, b, rest, out)
	case "cors":
		return cors(ctx, b, rest, out)
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}

// ---- push ----

var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// upload is one prepared local file and where it goes in the bucket.
type upload struct {
	label, path, key, contentType string
	size                          int64
}

func push(ctx context.Context, b *r2.Bucket, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	portraitIn := fs.String("portrait", "", "hand-made 9:16 cut for phones")
	name := fs.String("name", "", "name for this upload (default: from the file name)")
	posterAt := fs.Float64("poster-at", 0, "time in seconds of the poster frame")
	blurIn := fs.String("blur", "", "hand-made blurred still (JPEG) instead of a generated one")
	dryRun := fs.Bool("dry-run", false, "prepare everything but upload nothing")
	pos, err := parseInterleaved(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("push takes exactly one video\n\n" + usage)
	}
	in := pos[0]
	if *name == "" {
		*name = slug(strings.TrimSuffix(filepath.Base(in), filepath.Ext(in)))
	}
	if !validName.MatchString(*name) {
		return fmt.Errorf("name %q must be lowercase letters, digits and dashes", *name)
	}
	if err := media.CheckTools(); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "r2sync-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

	// Landscape cut.
	landscape := filepath.Join(tmp, "landscape.mp4")
	if err := prepareVideo(ctx, in, landscape, tw); err != nil {
		return err
	}
	lInfo, err := media.Probe(ctx, landscape)
	if err != nil {
		return err
	}
	if lInfo.Height > lInfo.Width {
		fmt.Fprintf(tw, "warning\t%s is taller than it is wide; pass the landscape cut first\n", filepath.Base(in))
	}

	// Portrait cut: supplied, or cropped from the middle of the landscape one.
	portrait := filepath.Join(tmp, "portrait.mp4")
	cropped := *portraitIn == ""
	if cropped {
		if err := media.CropPortrait(ctx, landscape, portrait); err != nil {
			return err
		}
		fmt.Fprintf(tw, "portrait\tcropped 9:16 from the middle (pass --portrait for a hand-framed cut)\n")
	} else if err := prepareVideo(ctx, *portraitIn, portrait, tw); err != nil {
		return err
	}
	pInfo, err := media.Probe(ctx, portrait)
	if err != nil {
		return err
	}

	// Stills.
	lPoster := filepath.Join(tmp, "poster.jpg")
	pPoster := filepath.Join(tmp, "portrait-poster.jpg")
	blur := filepath.Join(tmp, "blur.jpg")
	for _, step := range []func() error{
		func() error { return media.Poster(ctx, landscape, lPoster, *posterAt) },
		func() error { return media.Poster(ctx, portrait, pPoster, *posterAt) },
		func() error {
			if *blurIn != "" {
				return copyFile(*blurIn, blur)
			}
			return media.Blur(ctx, landscape, blur, *posterAt)
		},
	} {
		if err := step(); err != nil {
			return err
		}
	}

	uploads := []*upload{
		{label: "landscape", path: landscape, contentType: "video/mp4"},
		{label: "portrait", path: portrait, contentType: "video/mp4"},
		{label: "poster", path: lPoster, contentType: "image/jpeg"},
		{label: "portrait-poster", path: pPoster, contentType: "image/jpeg"},
		{label: "blur", path: blur, contentType: "image/jpeg"},
	}
	for _, u := range uploads {
		sum, size, err := hashFile(u.path)
		if err != nil {
			return err
		}
		u.size = size
		suffix := ""
		if u.label != "landscape" {
			suffix = "-" + u.label
		}
		u.key = fmt.Sprintf("%s%s%s-%s%s", mediaPrefix, *name, suffix, sum, filepath.Ext(u.path))
	}

	if *dryRun {
		for _, u := range uploads {
			fmt.Fprintf(tw, "would upload\t%s\t%s\n", u.key, humanBytes(u.size))
		}
		tw.Flush()
		fmt.Fprintln(out, "dry run: nothing uploaded, manifest unchanged")
		return nil
	}

	for _, u := range uploads {
		exists, err := b.Exists(ctx, u.key)
		if err != nil {
			return err
		}
		if exists {
			fmt.Fprintf(tw, "upload\t%s\talready there, skipped\n", u.key)
			continue
		}
		if err := b.PutFile(ctx, u.key, u.path, u.contentType, r2.CacheImmutable); err != nil {
			return err
		}
		fmt.Fprintf(tw, "upload\t%s\t%s ✓\n", u.key, humanBytes(u.size))
	}

	m, err := loadManifest(ctx, b, false)
	if err != nil {
		return err
	}
	previous := m.Current
	now := time.Now().UTC()
	replaced := m.Put(*name, &manifest.Video{
		Landscape:       manifest.Cut{Src: uploads[0].key, Poster: uploads[2].key, Width: lInfo.Width, Height: lInfo.Height, Bytes: uploads[0].size},
		Portrait:        manifest.Cut{Src: uploads[1].key, Poster: uploads[3].key, Width: pInfo.Width, Height: pInfo.Height, Bytes: uploads[1].size},
		Blur:            uploads[4].key,
		Uploaded:        now,
		PortraitCropped: cropped,
	}, now)
	if err := b.PutJSON(ctx, manifest.Key, m, r2.CacheManifest); err != nil {
		return err
	}
	note := ""
	switch {
	case replaced:
		note = fmt.Sprintf(" (replaced the earlier %q upload; run `r2sync prune` to delete its files)", *name)
	case previous != "" && previous != *name:
		note = fmt.Sprintf(" (previous: %s; roll back with `r2sync use %s`)", previous, previous)
	}
	fmt.Fprintf(tw, "manifest\t%s is live%s\n", *name, note)
	tw.Flush()
	if u := b.PublicURL(manifest.Key); u != "" {
		fmt.Fprintf(out, "\n%s\n", u)
	}
	return nil
}

// prepareVideo runs faststart on one input and reports what it found.
func prepareVideo(ctx context.Context, in, out string, tw io.Writer) error {
	if _, err := os.Stat(in); err != nil {
		return err
	}
	wasFast, err := media.IsFaststart(in)
	if err != nil {
		return fmt.Errorf("%s: %w", in, err)
	}
	if err := media.Faststart(ctx, in, out); err != nil {
		return err
	}
	info, err := media.Probe(ctx, out)
	if err != nil {
		return err
	}
	st, err := os.Stat(out)
	if err != nil {
		return err
	}
	state := "moved index to front"
	if wasFast {
		state = "already faststart"
	}
	fmt.Fprintf(tw, "%s\t%dx%d  %s\t%s ✓\n", filepath.Base(in), info.Width, info.Height, humanBytes(st.Size()), state)
	return nil
}

// ---- list / use / remove / prune / cors ----

func list(ctx context.Context, b *r2.Bucket, out io.Writer) error {
	m, err := loadManifest(ctx, b, true)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "\tNAME\tUPLOADED\tLANDSCAPE\tPORTRAIT")
	for _, n := range m.Names() {
		v := m.Videos[n]
		mark := ""
		if n == m.Current {
			mark = "*"
		}
		portrait := fmt.Sprintf("%dx%d  %s", v.Portrait.Width, v.Portrait.Height, humanBytes(v.Portrait.Bytes))
		if v.PortraitCropped {
			portrait += "  (cropped)"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%dx%d  %s\t%s\n", mark, n, v.Uploaded.Local().Format("2006-01-02 15:04"),
			v.Landscape.Width, v.Landscape.Height, humanBytes(v.Landscape.Bytes), portrait)
	}
	return tw.Flush()
}

func use(ctx context.Context, b *r2.Bucket, args []string, out io.Writer) error {
	if len(args) != 1 {
		return errors.New("use takes one name; see `r2sync list`")
	}
	m, err := loadManifest(ctx, b, true)
	if err != nil {
		return err
	}
	if m.Current == args[0] {
		fmt.Fprintf(out, "%s is already live\n", args[0])
		return nil
	}
	previous := m.Current
	if err := m.Use(args[0], time.Now().UTC()); err != nil {
		return err
	}
	if err := b.PutJSON(ctx, manifest.Key, m, r2.CacheManifest); err != nil {
		return err
	}
	fmt.Fprintf(out, "%s is live (was %s)\n", args[0], previous)
	return nil
}

func remove(ctx context.Context, b *r2.Bucket, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "actually delete")
	pos, err := parseInterleaved(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return errors.New("remove takes one name; see `r2sync list`")
	}
	name := pos[0]
	m, err := loadManifest(ctx, b, true)
	if err != nil {
		return err
	}
	v, ok := m.Videos[name]
	if !ok {
		return fmt.Errorf("no video named %q (have: %v)", name, m.Names())
	}
	if err := m.Remove(name, time.Now().UTC()); err != nil {
		return err
	}
	// Only delete files no remaining upload shares (identical re-uploads
	// produce the same content-addressed keys).
	still := m.Referenced()
	var doomed []string
	for _, k := range v.Keys() {
		if k != "" && !still[k] {
			doomed = append(doomed, k)
		}
	}
	if !*yes {
		fmt.Fprintf(out, "would remove %q and delete:\n", name)
		for _, k := range doomed {
			fmt.Fprintf(out, "  %s\n", k)
		}
		fmt.Fprintln(out, "re-run with --yes to delete")
		return nil
	}
	// Manifest first, so the site never points at a deleted file.
	if err := b.PutJSON(ctx, manifest.Key, m, r2.CacheManifest); err != nil {
		return err
	}
	for _, k := range doomed {
		if err := b.Delete(ctx, k); err != nil {
			return err
		}
		fmt.Fprintf(out, "deleted %s\n", k)
	}
	fmt.Fprintf(out, "removed %q\n", name)
	return nil
}

func prune(ctx context.Context, b *r2.Bucket, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("prune", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "actually delete")
	if _, err := parseInterleaved(fs, args); err != nil {
		return err
	}
	m, err := loadManifest(ctx, b, false)
	if err != nil {
		return err
	}
	objs, err := b.List(ctx, mediaPrefix)
	if err != nil {
		return err
	}
	refs := m.Referenced()
	var orphans []r2.Object
	var total int64
	for _, o := range objs {
		if !refs[o.Key] {
			orphans = append(orphans, o)
			total += o.Size
		}
	}
	if len(orphans) == 0 {
		fmt.Fprintln(out, "nothing to prune")
		return nil
	}
	sort.Slice(orphans, func(i, j int) bool { return orphans[i].Key < orphans[j].Key })
	verb := "would delete"
	if *yes {
		verb = "deleted"
	}
	for _, o := range orphans {
		if *yes {
			if err := b.Delete(ctx, o.Key); err != nil {
				return err
			}
		}
		fmt.Fprintf(out, "%s %s  %s\n", verb, o.Key, humanBytes(o.Size))
	}
	fmt.Fprintf(out, "%d file(s), %s\n", len(orphans), humanBytes(total))
	if !*yes {
		fmt.Fprintln(out, "re-run with --yes to delete")
	}
	return nil
}

// defaultOrigins are the places the site runs: production, the Vite dev
// server and `vite preview`.
var defaultOrigins = []string{"https://nickanderson.cloud", "http://localhost:5173", "http://localhost:4173"}

func cors(ctx context.Context, b *r2.Bucket, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("cors", flag.ContinueOnError)
	var origins multiFlag
	fs.Var(&origins, "origin", "allowed origin (repeatable; default: production + local dev)")
	if _, err := parseInterleaved(fs, args); err != nil {
		return err
	}
	if len(origins) == 0 {
		origins = defaultOrigins
	}
	if err := b.SetCORS(ctx, origins); err != nil {
		return err
	}
	fmt.Fprintf(out, "CORS set: GET/HEAD from %s\n", strings.Join(origins, ", "))
	return nil
}

// ---- helpers ----

// loadManifest reads manifest.json. With mustExist, a missing manifest is an
// error (nothing to list or switch to yet); otherwise it starts empty.
func loadManifest(ctx context.Context, b *r2.Bucket, mustExist bool) (*manifest.Manifest, error) {
	m := manifest.New()
	found, err := b.GetJSON(ctx, manifest.Key, m)
	if err != nil {
		return nil, err
	}
	if !found && mustExist {
		return nil, errors.New("no manifest in the bucket yet; run `r2sync push <video>` first")
	}
	if m.Videos == nil {
		m.Videos = map[string]*manifest.Video{}
	}
	return m, nil
}

// parseInterleaved parses flags that may appear before, between or after
// positional arguments (Go's flag package stops at the first positional).
func parseInterleaved(fs *flag.FlagSet, args []string) ([]string, error) {
	fs.SetOutput(io.Discard)
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, fmt.Errorf("%s: %w", fs.Name(), err)
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

func copyFile(src, dst string) error {
	if !strings.EqualFold(filepath.Ext(src), ".jpg") && !strings.EqualFold(filepath.Ext(src), ".jpeg") {
		return fmt.Errorf("--blur must be a JPEG: %s", src)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func hashFile(path string) (sum string, size int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	size, err = io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil))[:10], size, nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func humanBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }
