import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { execSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

const repoRoot = fileURLToPath(new URL('..', import.meta.url));

// Product version is the git tag (e.g. v1.0.0). Root package.json is not a release source.
function gitDescribe() {
	try {
		return execSync('git describe --tags --always', {
			cwd: repoRoot,
			stdio: ['ignore', 'pipe', 'ignore']
		})
			.toString()
			.trim()
			.replace(/^v/, '');
	} catch {
		return '';
	}
}

const fromEnv = process.env.APP_VERSION?.trim().replace(/^v/, '') ?? '';
const appVersion = fromEnv || gitDescribe() || 'dev';

export default defineConfig({
	plugins: [
		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/lib/paraglide',
			strategy: ['cookie', 'preferredLanguage', 'baseLocale'],
			emitTsDeclarations: true,
			// Pinned so dev, build and the i18n:compile script (npm test / check) all write the
			// same layout into src/lib/paraglide. Otherwise the plugin uses locale-modules in dev
			// and message-modules in production, and a compile while `vite dev` runs leaves the
			// dev server importing files that no longer exist (404 on messages/<key>.js).
			outputStructure: 'message-modules'
		}),
		tailwindcss(),
		sveltekit()
	],
	define: { __APP_VERSION__: JSON.stringify(appVersion) }
});
