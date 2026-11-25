import { useState, useSyncExternalStore } from 'react';
import { PORTRAIT_QUERY, type WarpParams } from '../warp/engine';
import styles from './WarpPanel.module.css';

type Slider = { name: keyof WarpParams; min: number; max: number; step: number };

const SLIDERS: Slider[] = [
	{ name: 'zoom', min: 1, max: 2, step: 0.01 },
	{ name: 'amplitude', min: 0, max: 0.25, step: 0.005 },
	{ name: 'size', min: 0.3, max: 6, step: 0.1 },
	{ name: 'speed', min: 0, max: 0.4, step: 0.005 },
	{ name: 'ripple', min: 0, max: 0.03, step: 0.0005 },
	{ name: 'vignette', min: 0, max: 1, step: 0.01 },
	{ name: 'tint', min: 0, max: 1, step: 0.01 },
];

const OPEN_KEY = 'vwarp-panel-open';

function usePortrait() {
	return useSyncExternalStore(
		(cb) => {
			const mq = matchMedia(PORTRAIT_QUERY);
			mq.addEventListener('change', cb);
			return () => mq.removeEventListener('change', cb);
		},
		() => matchMedia(PORTRAIT_QUERY).matches,
	);
}

function initialOpen() {
	// Collapsed by default on phones; remember the choice per browser.
	try {
		const saved = localStorage.getItem(OPEN_KEY);
		if (saved !== null) return saved === '1';
	} catch {}
	return matchMedia('(min-width: 601px)').matches;
}

// Tuning panel (dev only). Edits the live params object the render loop reads.
// The zoom slider edits whichever zoom matches the current screen shape, so
// landscape and portrait can be tuned separately.
export default function WarpPanel({ params, onVignette }: { params: WarpParams; onVignette: (v: number) => void }) {
	const portrait = usePortrait();
	const [values, setValues] = useState<WarpParams>(() => ({ ...params }));
	const [open, setOpen] = useState(initialOpen);

	const toggle = () => {
		const next = !open;
		setOpen(next);
		try { localStorage.setItem(OPEN_KEY, next ? '1' : '0'); } catch {}
	};

	const update = (name: Slider['name'], value: number) => {
		const key: keyof WarpParams = name === 'zoom' && portrait ? 'portraitZoom' : name;
		params[key] = value;
		if (key === 'vignette') onVignette(value);
		setValues((v) => ({ ...v, [key]: value }));
	};

	const output = Object.entries(values)
		.filter(([k]) => k !== 'blur')
		.map(([k, v]) => `${k}={${v}}`)
		.join('\n');

	return (
		<div className={styles.panel}>
			<button type="button" className={styles.toggle} aria-expanded={open} aria-controls="vwarp-body" onClick={toggle}>
				warp
			</button>
			<form className={styles.body} id="vwarp-body" aria-label="Warp controls" onSubmit={(e) => e.preventDefault()}>
				{SLIDERS.map(({ name, min, max, step }) => {
					const isZoom = name === 'zoom';
					const value = isZoom && portrait ? values.portraitZoom : values[name];
					return (
						<label key={name}>
							<span>{isZoom && portrait ? 'zoom (portrait)' : name}</span>{' '}
							<input type="range" name={name} min={min} max={max} step={step} value={value} onChange={(e) => update(name, +e.target.value)} />
						</label>
					);
				})}
				<output>{output}</output>
			</form>
		</div>
	);
}
