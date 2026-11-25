import { FRAG, VERT } from './shaders';

export interface WarpParams {
	zoom: number; // >1 crops in so warped edges never show
	portraitZoom: number; // the portrait cut needs much less crop
	amplitude: number; // strength of the big slow warp
	size: number; // scale of the big warp blobs (higher = smaller blobs)
	speed: number; // how fast the warp drifts
	ripple: number; // strength of the fine ripple
	tint: number; // 0..1 darkening toward --ink for text legibility
	blur: number; // soft-focus radius in CSS pixels (0 = sharp)
	vignette: number; // 0..1 darkening toward the edges and corners
}

export interface WarpSources {
	src: string;
	poster: string;
	portraitSrc: string;
	portraitPoster: string;
}

export interface WarpOptions {
	root: HTMLElement;
	video: HTMLVideoElement;
	canvas: HTMLCanvasElement;
	sources: WarpSources;
	/** Read every frame, so the dev panel can mutate it live. */
	params: WarpParams;
	/** Class added to root once WebGL is drawing (hides the CSS fallback). */
	glClass: string;
}

export const PORTRAIT_QUERY = '(max-aspect-ratio: 1/1)';

// requestVideoFrameCallback isn't in every TS DOM lib yet.
type RVFCVideo = HTMLVideoElement & {
	requestVideoFrameCallback(cb: () => void): number;
	cancelVideoFrameCallback(handle: number): void;
};

function hexToRgb(hex: string): [number, number, number] {
	const m = hex.trim().replace('#', '');
	const n = parseInt(m.length === 3 ? m.replace(/./g, (c) => c + c) : m, 16);
	return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
}

/**
 * Starts the background: picks the cut for the screen shape, plays it, and
 * (where WebGL exists) warps it through the shader. Without WebGL the plain
 * <video> stays visible; with reduced motion or data saver only the poster
 * shows and the video never loads.
 *
 * Returns a teardown that stops playback, drops every listener and frees the
 * GL objects, so the component can unmount on navigation (and survive React
 * StrictMode's mount/unmount/mount in dev).
 */
export function startWarp({ root, video, canvas, sources, params, glClass }: WarpOptions): () => void {
	const cleanups: Array<() => void> = [];
	let disposed = false;
	const on = <T extends EventTarget>(target: T, type: string, fn: EventListener) => {
		target.addEventListener(type, fn);
		cleanups.push(() => target.removeEventListener(type, fn));
	};

	const portraitMq = matchMedia(PORTRAIT_QUERY);
	const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
	const saveData = (navigator as Navigator & { connection?: { saveData?: boolean } }).connection?.saveData === true;
	const isPortrait = () => portraitMq.matches;
	const activeZoom = () => (isPortrait() ? params.portraitZoom : params.zoom);

	// ?debug: on-screen readout of what the browser does with playback.
	const debug = /[?&]debug\b/.test(location.search) ? makeDebug(video, root, glClass, { reduced, saveData }) : null;
	if (debug) cleanups.push(debug.dispose);

	// Autoplay can be refused (e.g. iOS Low Power Mode): retry on first touch.
	let retry: (() => void) | null = null;
	const disarmRetry = () => {
		if (!retry) return;
		document.removeEventListener('pointerdown', retry);
		document.removeEventListener('touchend', retry);
		retry = null;
	};
	cleanups.push(disarmRetry);

	const play = () => {
		if (disposed) return;
		debug?.log('play() requested');
		video.play().then(() => debug?.log('play() OK')).catch((err) => {
			debug?.log(`play() REFUSED: ${err?.name}: ${err?.message}`);
			if (retry || disposed) return;
			retry = () => {
				disarmRetry();
				video.play().catch(() => {});
			};
			document.addEventListener('pointerdown', retry);
			document.addEventListener('touchend', retry);
		});
	};

	// Pick the cut that matches the screen's shape.
	const applySource = () => {
		const portrait = isPortrait();
		const poster = portrait ? sources.portraitPoster : sources.poster;
		if (poster && video.getAttribute('poster') !== poster) video.poster = poster;
		if (reduced || saveData) return; // still poster only
		const src = portrait ? sources.portraitSrc : sources.src;
		if (video.getAttribute('src') !== src) {
			// iOS needs the muted *property* set before playback is requested.
			video.muted = true;
			video.defaultMuted = true;
			video.playsInline = true;
			video.autoplay = true;
			video.crossOrigin = 'anonymous';
			const t = video.currentTime;
			video.src = src;
			video.currentTime = t;
			play();
		}
	};

	applySource();
	on(portraitMq, 'change', applySource);
	// If the browser pauses the background on its own (power saving, media
	// session changes), quietly try to resume while the page is visible.
	on(video, 'pause', () => {
		if (disposed || document.hidden || reduced || saveData) return;
		setTimeout(() => { if (!disposed && video.paused && !document.hidden) play(); }, 250);
	});

	// Stop the network and decoder when unmounting. Runs last (cleanups run in
	// reverse), after `disposed` is set so the pause handler above stays quiet.
	cleanups.unshift(() => {
		video.pause();
		video.removeAttribute('src');
		video.load();
	});

	const dispose = () => {
		disposed = true;
		for (let i = cleanups.length - 1; i >= 0; i--) cleanups[i]();
		cleanups.length = 0;
	};

	if (reduced || saveData) return dispose;

	const gl = canvas.getContext('webgl', { antialias: false, premultipliedAlpha: false });
	if (!gl) return dispose; // plain <video> stays visible

	const shaders: WebGLShader[] = [];
	const compile = (type: number, src: string) => {
		const s = gl.createShader(type)!;
		gl.shaderSource(s, src);
		gl.compileShader(s);
		if (!gl.getShaderParameter(s, gl.COMPILE_STATUS)) throw new Error(gl.getShaderInfoLog(s) || 'shader');
		shaders.push(s);
		return s;
	};
	const prog = gl.createProgram()!;
	gl.attachShader(prog, compile(gl.VERTEX_SHADER, VERT));
	gl.attachShader(prog, compile(gl.FRAGMENT_SHADER, FRAG));
	gl.linkProgram(prog);
	gl.useProgram(prog);

	// One oversized triangle covers the screen.
	const buf = gl.createBuffer();
	gl.bindBuffer(gl.ARRAY_BUFFER, buf);
	gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
	const loc = gl.getAttribLocation(prog, 'p');
	gl.enableVertexAttribArray(loc);
	gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);

	const tex = gl.createTexture();
	gl.bindTexture(gl.TEXTURE_2D, tex);
	gl.pixelStorei(gl.UNPACK_FLIP_Y_WEBGL, true);
	// Video sizes are rarely powers of two: clamp + linear, no mipmaps.
	gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE);
	gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE);
	gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
	gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);

	// The canvas outlives a StrictMode remount, and getContext() hands back the
	// same context, so free what we made instead of losing the context.
	cleanups.push(() => {
		gl.deleteTexture(tex);
		gl.deleteBuffer(buf);
		gl.deleteProgram(prog);
		shaders.forEach((s) => gl.deleteShader(s));
		root.classList.remove(glClass);
	});

	const u = (n: string) => gl.getUniformLocation(prog, n);
	const U = {
		time: u('uTime'), zoom: u('uZoom'), amp: u('uAmp'), size: u('uSize'), speed: u('uSpeed'),
		ripple: u('uRipple'), tint: u('uTint'), cover: u('uCover'), ink: u('uInk'), blur: u('uBlur'), vignette: u('uVignette'),
	};
	const ink = getComputedStyle(document.documentElement).getPropertyValue('--ink') || '#141830';
	gl.uniform3fv(U.ink, hexToRgb(ink));

	let hasFrame = false;
	let videoLive = false; // true once real video frames are arriving
	let texW = 16, texH = 9; // size of whatever is in the texture, for cover math
	const upload = () => {
		if (video.readyState < 2 || video.paused) return;
		gl.bindTexture(gl.TEXTURE_2D, tex);
		gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, video);
		texW = video.videoWidth || texW;
		texH = video.videoHeight || texH;
		hasFrame = true;
		if (!videoLive) { videoLive = true; debug?.log('texture: live video'); }
	};

	// Warp the poster still until real frames arrive. Covers slow networks
	// and browsers that refuse autoplay outright (iOS Low Power Mode, Brave,
	// etc.): the background still flows instead of sitting frozen.
	let posterSrc = '';
	const loadPoster = () => {
		const src = video.poster;
		if (!src || src === posterSrc || videoLive) return;
		posterSrc = src;
		const img = new Image();
		img.crossOrigin = 'anonymous'; // cross-origin bucket: required to upload it to WebGL
		img.onload = () => {
			if (disposed || videoLive || img.src !== new URL(posterSrc, location.href).href) return;
			gl.bindTexture(gl.TEXTURE_2D, tex);
			gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, img);
			texW = img.naturalWidth;
			texH = img.naturalHeight;
			hasFrame = true;
			debug?.log('texture: poster still (warping)');
		};
		img.src = src;
	};
	loadPoster();
	on(portraitMq, 'change', () => { if (!videoLive) loadPoster(); });

	// Upload only when the video actually has a new frame, where supported.
	const rvfc = 'requestVideoFrameCallback' in video;
	if (rvfc) {
		const v = video as RVFCVideo;
		let handle = 0;
		const onFrame = () => { upload(); handle = v.requestVideoFrameCallback(onFrame); };
		handle = v.requestVideoFrameCallback(onFrame);
		cleanups.push(() => v.cancelVideoFrameCallback(handle));
	}

	// Size the drawing buffer to the element, and only reallocate when that
	// actually changes. Phones have 3x screens; 1.5x is plenty under a warp.
	const resize = () => {
		const dpr = Math.min(devicePixelRatio || 1, 1.5);
		const w = Math.round(canvas.clientWidth * dpr);
		const h = Math.round(canvas.clientHeight * dpr);
		if (w === canvas.width && h === canvas.height) return;
		canvas.width = w;
		canvas.height = h;
		gl.viewport(0, 0, w, h);
	};
	resize();
	const ro = new ResizeObserver(resize);
	ro.observe(canvas);
	cleanups.push(() => ro.disconnect());

	let raf = 0;
	const start = performance.now();
	const draw = (now: number) => {
		if (!rvfc) upload();
		if (hasFrame) {
			// object-fit: cover math
			const va = texW / texH;
			const ca = canvas.width / canvas.height;
			const cover = ca > va ? [1, va / ca] : [ca / va, 1];
			gl.uniform2f(U.cover, cover[0], cover[1]);
			// Blur is set in CSS px; convert to texture UV units for this frame.
			const dpr = canvas.width / (canvas.clientWidth || canvas.width);
			const z = activeZoom();
			gl.uniform2f(U.blur,
				(params.blur * dpr / canvas.width) * cover[0] / z,
				(params.blur * dpr / canvas.height) * cover[1] / z);
			gl.uniform1f(U.time, (now - start) / 1000);
			gl.uniform1f(U.zoom, z);
			gl.uniform1f(U.amp, params.amplitude);
			gl.uniform1f(U.size, params.size);
			gl.uniform1f(U.speed, params.speed);
			gl.uniform1f(U.ripple, params.ripple);
			gl.uniform1f(U.tint, params.tint);
			gl.uniform1f(U.vignette, params.vignette);
			gl.drawArrays(gl.TRIANGLES, 0, 3);
			root.classList.add(glClass);
		}
		raf = requestAnimationFrame(draw);
	};
	raf = requestAnimationFrame(draw);
	cleanups.push(() => cancelAnimationFrame(raf));

	// Pause everything while the tab is hidden.
	on(document, 'visibilitychange', () => {
		if (document.hidden) { cancelAnimationFrame(raf); video.pause(); }
		else { play(); cancelAnimationFrame(raf); raf = requestAnimationFrame(draw); }
	});

	return dispose;
}

function makeDebug(video: HTMLVideoElement, root: HTMLElement, glClass: string, flags: Record<string, boolean>) {
	const box = document.createElement('pre');
	box.style.cssText = 'position:fixed;top:8px;left:8px;right:8px;z-index:9999;margin:0;padding:8px 10px;font:11px/1.35 ui-monospace,monospace;color:#0f0;background:rgba(0,0,0,.8);border-radius:6px;white-space:pre-wrap;pointer-events:none;max-height:45vh;overflow:hidden';
	document.body.appendChild(box);
	const events: string[] = [];
	const t0 = performance.now();
	const log = (m: string) => { events.push(`${((performance.now() - t0) / 1000).toFixed(1)}s ${m}`); if (events.length > 12) events.shift(); };
	const types = ['loadstart', 'loadeddata', 'canplay', 'playing', 'pause', 'waiting', 'stalled', 'suspend', 'error'];
	const handlers = types.map((e) => {
		const fn = () => log(`event: ${e}${e === 'error' ? ' ' + (video.error?.message || video.error?.code) : ''}`);
		video.addEventListener(e, fn);
		return [e, fn] as const;
	});
	let raf = 0;
	const render = () => {
		box.textContent = [
			navigator.userAgent.replace(/^Mozilla\/5\.0 /, '').slice(0, 120),
			`flags: ${JSON.stringify(flags)}  webgl: ${root.classList.contains(glClass)}`,
			`src: ${video.currentSrc.split('/').pop() || '(none)'}  muted: ${video.muted}  playsInline: ${video.playsInline}`,
			`paused: ${video.paused}  time: ${video.currentTime.toFixed(2)}  readyState: ${video.readyState}  visibility: ${getComputedStyle(video).visibility}`,
			'---',
			...events,
		].join('\n');
		raf = requestAnimationFrame(render);
	};
	render();
	const dispose = () => {
		cancelAnimationFrame(raf);
		handlers.forEach(([e, fn]) => video.removeEventListener(e, fn));
		box.remove();
	};
	return { log, dispose };
}
