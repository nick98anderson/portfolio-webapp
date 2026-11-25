import type { CSSProperties } from 'react';
import styles from './StillBackdrop.module.css';

// Calm backdrop for reading pages: a heavily blurred, darkened frame of the
// homepage video. Shares a view-transition name with VideoWarp so navigating
// between them crossfades the live video into the still.
export default function StillBackdrop({ src }: { src?: string }) {
	// Until the manifest loads, keep the gradients over plain ink.
	const still = src ? `url('${src}')` : 'none';
	return <div className={styles.still} aria-hidden="true" style={{ '--still': still } as CSSProperties} />;
}
