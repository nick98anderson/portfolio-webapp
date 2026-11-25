import { copyFile, mkdir } from 'node:fs/promises';
import { join } from 'node:path';
import react from '@vitejs/plugin-react';
import { defineConfig, type Plugin } from 'vite';

// Client-side routes. GitHub Pages only serves files that exist, so each
// route gets a copy of index.html (served with a 200), and anything else
// falls back to 404.html, which also boots the app.
const routes = ['resume', 'about'];

function spaFallback(): Plugin {
	let outDir = 'dist';
	return {
		name: 'spa-fallback',
		apply: 'build',
		configResolved(config) {
			outDir = config.build.outDir;
		},
		async closeBundle() {
			const index = join(outDir, 'index.html');
			for (const route of routes) {
				await mkdir(join(outDir, route), { recursive: true });
				await copyFile(index, join(outDir, route, 'index.html'));
			}
			await copyFile(index, join(outDir, '404.html'));
		},
	};
}

export default defineConfig({
	plugins: [react(), spaFallback()],
	css: {
		modules: { localsConvention: 'camelCaseOnly' },
	},
});
