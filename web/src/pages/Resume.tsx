import type { CSSProperties } from 'react';
import Crumbs from '../components/Crumbs';
import StillBackdrop from '../components/StillBackdrop';
import { asset } from '../lib/asset';
import styles from './Resume.module.css';

// Stagger index for the reveal-on-load animation.
const i = (n: number) => ({ '--i': n }) as CSSProperties;

export default function Resume() {
	return (
		<>
			<StillBackdrop src={asset('video/bg-blur.jpg')} />
			<Crumbs />
			<main className={styles.resume}>
				<header className={`${styles.resumeHeader} ${styles.reveal}`} style={i(0)}>
					<img className={styles.avatar} src={asset('me.jpeg')} alt="Nick Anderson" width="132" height="132" />
					<h1 className={styles.name}>Nick Anderson</h1>
					<p className={styles.contact}>
						Calgary, Alberta, Canada&nbsp;&nbsp;•&nbsp;&nbsp;+1 (587) 982-0953&nbsp;&nbsp;•&nbsp;&nbsp;{' '}
						<a href="mailto:ncainanderson@gmail.com">ncainanderson@gmail.com</a>
					</p>
					<div className={styles.bio}>
						<p>
							I'm a data and software engineer at TELUS. Most of my time has gone into
							building out cloud-native data warehousing and analytics in GCP, and
							developing internal applications and automation tools for finance teams,
							often eliminating or avoiding the need for expensive third-party software.
						</p>
					</div>
				</header>

				<section className={styles.reveal} style={i(1)}>
					<h2 className={styles.sectionTitle}>Skills</h2>
					<dl className={styles.skills}>
						<dt>Languages</dt>
						<dd><span className={styles.lvl}>Proficient:</span> Python, SQL, JavaScript, TypeScript<span className={styles.lvlSep}>;</span> <span className={styles.lvl}>Familiar:</span> Go, Java, C++</dd>
						<dt>Technologies</dt>
						<dd><span className={styles.lvl}>Proficient:</span> GCP, BigQuery, dbt, Django, Next.js, React<span className={styles.lvlSep}>;</span> <span className={styles.lvl}>Familiar:</span> Kubernetes, Airflow</dd>
						<dt>Tools</dt>
						<dd>Git, GitHub Actions, Docker, Kubectl, Pulumi, Postman, gcloud CLI, Claude Code</dd>
						<dt>Non-Technical</dt>
						<dd>Teamwork, listening, leadership, public speaking, creative</dd>
					</dl>
				</section>

				<section className={styles.reveal} style={i(2)}>
					<h2 className={styles.sectionTitle}>Relevant Experience</h2>

					<article className={styles.role}>
						<div className={styles.roleHead}>
							<h3>TELUS <span className={styles.sep}>·</span> <span className={styles.roleTitle}>Data Engineer</span></h3>
							<span className={styles.meta}>Jul 2025 – Present</span>
						</div>
						<ul className={styles.roleSummary}>
							<li>Rebuilt a billing-to-ERP reconciliation pipeline in dbt across 8 billing systems, correcting faulty logic to eliminate $10M in revenue variances and remove manual reconciliation for finance teams.</li>
							<li>Engineered a master data web platform (Django, Next.js, GKE) to replace three legacy Access/VBA tools, automating data feeds into downstream ERPs while saving $240K/year in licensing.</li>
							<li>Eliminated licence-gated reporting by building extractions into BigQuery, replicating OLAP cube aggregations and deploying custom data apps and dashboards for 300+ self-service users.</li>
							<li>Built agentic harnesses with domain context and direct integrations into internal systems and data warehouses, used by a 5-person team to cut daily tasks and debugging from hours to minutes.</li>
						</ul>
					</article>

					<article className={styles.role}>
						<div className={styles.roleHead}>
							<h3>TELUS <span className={styles.sep}>·</span> <span className={styles.roleTitle}>Business Systems Analyst</span></h3>
							<span className={styles.meta}>May 2022 – Jul 2025</span>
						</div>
						<ul className={styles.roleSummary}>
							<li>Re-engineered an on-prem legacy data mart onto GCP (BigQuery, dbt, Python), eliminating proprietary ETL and retiring legacy infrastructure to cut $50K/year in licensing.</li>
							<li>Cut daily and month-end finance work from hours to minutes by replacing manual UI-data management and system configuration with automated REST API integrations and workflows.</li>
							<li>Modernized a traditionally low-code finance systems team by introducing version control and CI/CD, replacing fragile manual processes to cut routine toil and ship faster for internal clients.</li>
						</ul>
					</article>

					<article className={styles.role}>
						<div className={styles.roleHead}>
							<h3>Flare Analytics <span className={styles.sep}>·</span> <span className={styles.roleTitle}>Software Developer Intern</span></h3>
							<span className={styles.meta}>May 2018 – Sept 2018</span>
						</div>
						<ul className={styles.roleSummary}>
							<li>Developed client-facing analytics dashboards in React to visualize predictive oil and gas metrics.</li>
							<li>Worked alongside senior engineers on AWS deployment and release processes for client-facing web applications.</li>
						</ul>
					</article>
				</section>

				<section className={styles.reveal} style={i(3)}>
					<h2 className={styles.sectionTitle}>Selected Projects</h2>
					<div className={styles.project}>
						<div className={styles.projectHead}>
							<h4>Homebox <span className={styles.sep}>·</span> <span className={styles.roleTitle}>Home Automation Server</span></h4>
							<span className={styles.meta}>Oct 2026</span>
						</div>
						<ul>
							<li>Built a self-hosted media and home automation stack with Go background services handling TV control, lighting, and playback rules.</li>
							<li>Created a mobile web app and CLI to search, browse the library, and control the TV from anywhere, exposed remotely over Tailscale HTTPS with session auth.</li>
						</ul>
					</div>
				</section>

				<section className={styles.reveal} style={i(4)}>
					<h2 className={styles.sectionTitle}>Education</h2>
					<div className={styles.edu}>
						<h3>University of Alberta <span className={styles.sep}>·</span> <span className={styles.eduTitle}>BSc, Computing Science</span></h3>
						<span className={styles.meta}>Sep 2019 – Dec 2023</span>
					</div>
				</section>
			</main>
		</>
	);
}
