import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, relative, resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

/*
 * The product name is always "SOPandGO"; identifiers stay lowercase
 * "sopandgo" (docs/design/style-guide.md → Product name). A line that lists
 * wrong spellings on purpose must say "never" (as the style guide does).
 */

const repo = resolve(process.cwd(), '..');

// Any spelling of the name: split ("SOP and GO", "SOP & GO") or joined in any case.
const ANY_SPELLING = /sop\s*(?:and|&|\+)\s*go/gi;

function walk(dir: string, accept: (path: string) => boolean, out: string[] = []): string[] {
	for (const entry of readdirSync(dir)) {
		if (['node_modules', '.git', 'paraglide', 'tmp', 'build', '.svelte-kit', 'data'].includes(entry)) continue;
		const path = join(dir, entry);
		if (statSync(path).isDirectory()) walk(path, accept, out);
		else if (accept(path)) out.push(path);
	}
	return out;
}

/** Wrong spellings in free text; lowercase "sopandgo" and the GitHub org URL are identifiers. */
function wrongSpellings(text: string): string[] {
	const found: string[] = [];
	for (const line of text.split('\n')) {
		if (/\bnever\b/i.test(line)) continue;
		for (const m of line.matchAll(ANY_SPELLING)) {
			const word = m[0];
			if (word === 'SOPandGO' || word === 'sopandgo') continue;
			if (word === 'Sopandgo' && line[m.index + word.length] === '/') continue;
			found.push(word);
		}
	}
	return found;
}

describe('product name', () => {
	it('is SOPandGO in every message, in every language', () => {
		const catalogs = [
			...walk(join(repo, 'frontend/messages'), (p) => p.endsWith('.json')),
			...walk(join(repo, 'backend/internal/i18n/messages'), (p) => p.endsWith('.json'))
		];
		expect(catalogs.length).toBeGreaterThan(20);
		const problems: string[] = [];
		for (const file of catalogs) {
			const messages = JSON.parse(readFileSync(file, 'utf8')) as Record<string, string>;
			for (const [key, value] of Object.entries(messages)) {
				// Messages are prose only, so lowercase is wrong here too.
				for (const m of value.matchAll(ANY_SPELLING)) {
					if (m[0] !== 'SOPandGO') problems.push(`${relative(repo, file)} ${key}: "${m[0]}"`);
				}
			}
		}
		expect(problems).toEqual([]);
	});

	it('is never split or re-cased in code, config, or docs', () => {
		const self = resolve(process.cwd(), 'src/lib/productName.test.ts');
		const files = [
			...walk(join(repo, 'frontend/src'), (p) => /\.(ts|svelte)$/.test(p) && p !== self),
			// Go tests feed wrong spellings on purpose.
			...walk(join(repo, 'backend'), (p) => (p.endsWith('.go') && !p.endsWith('_test.go')) || p.endsWith('.md')),
			...walk(join(repo, 'docs'), (p) => /\.(md|json)$/.test(p)),
			...['README.md', 'CONTRIBUTING.md', 'SECURITY.md', 'CHANGELOG.md', 'AGENTS.md', 'Dockerfile', 'Caddyfile', 'frontend/README.md'].map(
				(f) => join(repo, f)
			)
		];
		const problems = files.flatMap((file) =>
			wrongSpellings(readFileSync(file, 'utf8')).map((word) => `${relative(repo, file)}: "${word}"`)
		);
		expect(problems).toEqual([]);
	});
});
