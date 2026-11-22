// @ts-check
import { defineConfig } from 'astro/config';

// https://astro.build/config
export default defineConfig({
	site: 'https://nickanderson.cloud',
	// About was folded into the resume page; keep old links working.
	redirects: {
		'/about': '/resume',
	},
});
