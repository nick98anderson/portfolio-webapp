// Public base URL of the R2 media bucket: background videos, posters and
// manifest.json, all published by `go run ./cmd/r2sync push`.
//
// This is Cloudflare's r2.dev development URL, which is rate-limited and not
// cached at the edge. Swap it for a custom domain (e.g. media.nickanderson.cloud)
// once the domain's DNS is on Cloudflare.
export const MEDIA_URL = 'https://pub-4fad8d633bcc41c0ae777108bf34ff5b.r2.dev';

export const MANIFEST_URL = `${MEDIA_URL}/manifest.json`;
