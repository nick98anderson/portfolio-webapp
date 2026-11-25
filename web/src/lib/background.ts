import { useEffect, useState } from 'react';
import { MANIFEST_URL, MEDIA_URL } from '../config';

// Which background video is live comes from manifest.json in the R2 bucket,
// so a new video goes live with `r2sync push`, no rebuild or deploy. The
// shape mirrors internal/manifest in the Go code.
interface Cut {
	src: string;
	poster: string;
}
interface Manifest {
	current: string;
	videos: Record<string, { landscape: Cut; portrait: Cut; blur: string }>;
}

/** Absolute URLs for the live background. */
export interface Background {
	src: string;
	poster: string;
	portraitSrc: string;
	portraitPoster: string;
	blur: string;
}

let pending: Promise<Background | null> | undefined;
let loaded: Background | null | undefined;

/**
 * Fetches the manifest once per page load (index.html preloads it, so this
 * usually resolves from that request). Resolves to null if it can't be
 * read: the page then shows the plain ink background instead of breaking.
 */
export function loadBackground(): Promise<Background | null> {
	pending ??= fetch(MANIFEST_URL)
		.then((res) => {
			if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
			return res.json() as Promise<Manifest>;
		})
		.then((m) => {
			const v = m.videos[m.current];
			if (!v) throw new Error(`current video "${m.current}" is not in the manifest`);
			const url = (key: string) => `${MEDIA_URL}/${key}`;
			return {
				src: url(v.landscape.src),
				poster: url(v.landscape.poster),
				portraitSrc: url(v.portrait.src),
				portraitPoster: url(v.portrait.poster),
				blur: url(v.blur),
			};
		})
		.catch((err: unknown) => {
			console.error(`Couldn't load the background manifest (${MANIFEST_URL}):`, err);
			return null;
		})
		.then((bg) => (loaded = bg));
	return pending;
}

/** The live background, or null until (or unless) the manifest loads. */
export function useBackground(): Background | null {
	const [bg, setBg] = useState(() => loaded ?? null);
	useEffect(() => {
		if (loaded !== undefined) return;
		let live = true;
		loadBackground().then((b) => live && setBg(b));
		return () => {
			live = false;
		};
	}, []);
	return bg;
}
