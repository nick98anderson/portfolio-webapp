# nickanderson.cloud

Personal site: a full-screen underwater video warped in real time by a WebGL
shader, plus a resume page.

## Layout

```text
web/                 React + Vite + TypeScript frontend
├── public/          static files copied as-is (videos, posters, CNAME)
└── src/
    ├── components/  Hero, VideoWarp (+ dev tuning panel), Crumbs, Grain, StillBackdrop
    ├── pages/       Home, Resume
    ├── warp/        WebGL engine and shaders behind VideoWarp
    └── router.tsx   routes: /, /resume, /about → /resume
```

## Commands

Run from `web/`:

| Command          | Action                                        |
| :--------------- | :-------------------------------------------- |
| `pnpm install`   | Install dependencies                          |
| `pnpm dev`       | Dev server at `localhost:5173` (with the warp tuning panel) |
| `pnpm build`     | Type-check and build to `web/dist/`           |
| `pnpm preview`   | Serve the production build locally            |
| `pnpm typecheck` | Type-check only                               |

Add `?debug` to the URL for an on-screen readout of video playback state.

## Deploy

Pushes to `main` build `web/` and publish `web/dist` to GitHub Pages
(`.github/workflows/deploy.yml`). Each client route also gets a copy of
`index.html` at build time so deep links like `/resume` load directly.
