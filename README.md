# nickanderson.cloud

Personal site: a full-screen underwater video warped in real time by a WebGL
shader, plus a resume page.

## Layout

```text
cmd/r2sync/          CLI that publishes background videos to Cloudflare R2
internal/            Go packages: manifest, media (ffmpeg), r2, dotenv
web/                 React + Vite + TypeScript frontend
├── public/          static files copied as-is (photo, CNAME)
└── src/
    ├── components/  Hero, VideoWarp (+ dev tuning panel), Crumbs, Grain, StillBackdrop
    ├── pages/       Home, Resume
    ├── warp/        WebGL engine and shaders behind VideoWarp
    ├── lib/         background manifest loader, asset paths
    ├── config.ts    media bucket URL
    └── router.tsx   routes: /, /resume, /about → /resume
```

## Background video (R2)

Videos, posters and the blurred still live in the `portfolio-media` R2
bucket, not in git. The site reads `manifest.json` from the bucket to find
the live video, so changing it needs no rebuild or deploy.

Credentials go in `.env` at the repo root (git-ignored):

```sh
R2_ACCOUNT_ID=…
R2_ACCESS_KEY_ID=…
R2_SECRET_ACCESS_KEY=…
R2_BUCKET=portfolio-media
R2_PUBLIC_URL=https://pub-….r2.dev
```

Publishing needs Go and ffmpeg (`brew install go ffmpeg`):

```sh
go run ./cmd/r2sync push ~/Videos/reef.mp4 --portrait ~/Videos/reef-portrait.mp4
go run ./cmd/r2sync list          # * marks the live video
go run ./cmd/r2sync use bg        # roll back to an earlier upload
go run ./cmd/r2sync remove reef   # delete an upload (asks for --yes)
go run ./cmd/r2sync prune         # delete files nothing points at (asks for --yes)
go run ./cmd/r2sync --help
```

`push` makes the MP4 faststart (index first, so playback starts while it
streams), drops any audio, grabs the first frame as the poster, makes the
blurred still (or takes yours with `--blur`), crops a 9:16 portrait cut if
you don't pass `--portrait`, uploads everything under content-hashed names
with a one-year cache, and points the manifest at it.

The bucket's CORS rules (needed for WebGL to read the video) are set with
`go run ./cmd/r2sync cors`; rerun it if the site's origin changes.

## Commands

Frontend, run from `web/`:

| Command          | Action                                        |
| :--------------- | :-------------------------------------------- |
| `pnpm install`   | Install dependencies                          |
| `pnpm dev`       | Dev server at `localhost:5173` (with the warp tuning panel) |
| `pnpm build`     | Type-check and build to `web/dist/`           |
| `pnpm preview`   | Serve the production build locally            |
| `pnpm typecheck` | Type-check only                               |

Go, run from the repo root: `go test ./...`.

Add `?debug` to the URL for an on-screen readout of video playback state.

## Deploy

Pushes to `main` build `web/` and publish `web/dist` to GitHub Pages
(`.github/workflows/deploy.yml`). Each client route also gets a copy of
`index.html` at build time so deep links like `/resume` load directly.
