import styles from './Grain.module.css';

// Print grain over everything: light specks plus dark specks, so it reads on
// both the dark backdrop and the cream type.
export default function Grain() {
	return (
		<>
			<div className={`${styles.grain} ${styles.light}`} aria-hidden="true" />
			<div className={`${styles.grain} ${styles.dark}`} aria-hidden="true" />
		</>
	);
}
