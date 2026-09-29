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
	plugins: [tailwindcss(), sveltekit()],
	define: {
		__APP_VERSION__: JSON.stringify(appVersion)
	}
});
