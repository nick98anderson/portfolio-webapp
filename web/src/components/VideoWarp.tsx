import { useEffect, useRef, type CSSProperties } from 'react';
import { startWarp, type WarpParams } from '../warp/engine';
import WarpPanel from './WarpPanel';
import styles from './VideoWarp.module.css';

// Full-screen background video, warped in real time by a WebGL shader:
// low-frequency noise slowly drags the picture around like liquid, with a
// finer ripple layered on top.
//
// - Portrait screens (phones) get their own 9:16 cut, so they show real
//   detail instead of an upscaled sliver of the landscape video.
// - Without WebGL the plain video plays with the same crop and darkening.
// - Reduced motion or data saver: a still poster, the video never loads.
interface Props extends Partial<WarpParams> {
	src: string;
	poster?: string;
	portraitSrc?: string; // 9:16 cut for portrait screens
	portraitPoster?: string;
	controls?: boolean; // show the tuning panel (dev only)
}

export default function VideoWarp({
	src,
	poster = '',
	portraitSrc = src,
	portraitPoster = poster,
	zoom = 1.35,
	portraitZoom = 1.06,
	amplitude = 0.06,
	size = 1.6,
	speed = 0.06,
	ripple = 0.004,
	tint = 0.45,
	blur = 26.5,
	vignette = 0.6,
	controls = false,
}: Props) {
	const rootRef = useRef<HTMLDivElement>(null);
	const videoRef = useRef<HTMLVideoElement>(null);
	const canvasRef = useRef<HTMLCanvasElement>(null);
	// Mutable on purpose: the render loop reads it every frame and the dev
	// panel writes to it, without re-rendering React.
	const params = useRef<WarpParams>({ zoom, portraitZoom, amplitude, size, speed, ripple, tint, blur, vignette }).current;

	useEffect(
		() =>
			startWarp({
				root: rootRef.current!,
				video: videoRef.current!,
				canvas: canvasRef.current!,
				sources: { src, poster, portraitSrc, portraitPoster },
				params,
				glClass: styles.gl,
			}),
		[src, poster, portraitSrc, portraitPoster, params],
	);

	const cssVars = {
		'--zoom': zoom,
		'--zoom-portrait': portraitZoom,
		'--tint': tint,
		'--blur': blur,
		'--vignette': vignette,
	} as CSSProperties;

	return (
		<>
			<div ref={rootRef} className={styles.root} aria-hidden="true" style={cssVars}>
				{/* src is set from JS so phones never download the landscape file. */}
				<video ref={videoRef} className={styles.video} poster={poster || undefined} muted loop playsInline preload="none" />
				<canvas ref={canvasRef} className={styles.canvas} />
			</div>
			{controls && (
				<WarpPanel params={params} onVignette={(v) => rootRef.current?.style.setProperty('--vignette', String(v))} />
			)}
		</>
	);
}
