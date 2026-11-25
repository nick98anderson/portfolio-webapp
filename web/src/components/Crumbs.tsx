import { NavLink } from 'react-router';
import styles from './Crumbs.module.css';

const activeClass = ({ isActive }: { isActive: boolean }) => (isActive ? styles.active : undefined);

export default function Crumbs() {
	return (
		<nav className={styles.crumbs} aria-label="Breadcrumb">
			<NavLink to="/" end viewTransition aria-label="Home" className={activeClass}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
					<path d="M3 11l9-8 9 8" />
					<path d="M5 10v10a1 1 0 0 0 1 1h4v-6h4v6h4a1 1 0 0 0 1-1V10" />
				</svg>
			</NavLink>
			<span className={styles.sep}>/</span>
			<NavLink to="/resume" viewTransition aria-label="Resume" className={activeClass}>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
					<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
					<polyline points="14 2 14 8 20 8" />
					<line x1="8" y1="13" x2="16" y2="13" />
					<line x1="8" y1="17" x2="16" y2="17" />
				</svg>
			</NavLink>
		</nav>
	);
}
