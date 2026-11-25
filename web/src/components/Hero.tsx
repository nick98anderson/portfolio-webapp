import { Link } from 'react-router';
import styles from './Hero.module.css';

// The intro animation replays whenever this route mounts, so coming back to
// the homepage (including the browser's back button) plays it again.
export default function Hero() {
	return (
		<section className={styles.hero}>
			<h1 className={styles.name}>Nick Anderson</h1>
			<div className={styles.below}>
				<span className={`${styles.rule} distress`} aria-hidden="true" />
				<nav className={styles.links} aria-label="Primary">
					<a href="https://www.linkedin.com/in/nicholas-anderson-a10617192/" target="_blank" rel="noopener" className={styles.link} aria-label="LinkedIn">
						<svg className={styles.icon} viewBox="0 0 24 24" aria-hidden="true">
							<path fill="currentColor" d="M20.45 20.45h-3.55v-5.57c0-1.33-.03-3.04-1.86-3.04-1.86 0-2.15 1.45-2.15 2.95v5.66H9.34V9h3.41v1.56h.05c.48-.9 1.64-1.86 3.38-1.86 3.61 0 4.27 2.38 4.27 5.47v6.28zM5.34 7.44a2.06 2.06 0 1 1 0-4.13 2.06 2.06 0 0 1 0 4.13zM7.11 20.45H3.56V9h3.55v11.45zM22.22 0H1.77C.79 0 0 .77 0 1.72v20.56C0 23.23.79 24 1.77 24h20.45C23.2 24 24 23.23 24 22.28V1.72C24 .77 23.2 0 22.22 0z" />
						</svg>
						<span>LinkedIn</span>
					</a>
					<a href="https://github.com/nick98anderson" target="_blank" rel="noopener" className={styles.link} aria-label="GitHub">
						<svg className={styles.icon} viewBox="0 0 24 24" aria-hidden="true">
							<path fill="currentColor" d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.55 0-.27-.01-1-.02-1.96-3.2.7-3.87-1.54-3.87-1.54-.53-1.34-1.29-1.7-1.29-1.7-1.05-.72.08-.71.08-.71 1.16.08 1.77 1.19 1.77 1.19 1.03 1.77 2.71 1.26 3.37.96.11-.75.4-1.26.73-1.55-2.55-.29-5.24-1.28-5.24-5.69 0-1.26.45-2.28 1.19-3.09-.12-.29-.52-1.47.11-3.06 0 0 .97-.31 3.18 1.18a11 11 0 0 1 5.79 0c2.21-1.49 3.18-1.18 3.18-1.18.63 1.59.23 2.77.11 3.06.74.81 1.19 1.83 1.19 3.09 0 4.42-2.7 5.4-5.27 5.68.41.35.78 1.05.78 2.11 0 1.53-.01 2.76-.01 3.13 0 .3.21.67.8.55C20.22 21.38 23.5 17.08 23.5 12 23.5 5.65 18.35.5 12 .5z" />
						</svg>
						<span>GitHub</span>
					</a>
					<Link to="/resume" viewTransition className={styles.link} aria-label="Resume">
						<svg className={styles.icon} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
							<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
							<polyline points="14 2 14 8 20 8" />
							<line x1="8" y1="13" x2="16" y2="13" />
							<line x1="8" y1="17" x2="16" y2="17" />
							<line x1="8" y1="9" x2="10" y2="9" />
						</svg>
						<span>Resume</span>
					</Link>
				</nav>
			</div>
		</section>
	);
}
