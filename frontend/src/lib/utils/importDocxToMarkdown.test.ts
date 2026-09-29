/** @vitest-environment jsdom */
import { describe, expect, it } from 'vitest';
import { mammothHtmlToSopMarkdown } from './importDocxToMarkdown';

describe('mammothHtmlToSopMarkdown', () => {
	it('does not leave raw <table> HTML when the table has no GFM header row', () => {
		const html = `
			<p>Intro</p>
			<table><tbody>
				<tr><td>One</td><td>Two</td></tr>
				<tr><td>Three</td><td>Four</td></tr>
			</tbody></table>
			<p>Outro</p>
		`;
		const md = mammothHtmlToSopMarkdown(html);
		expect(md.toLowerCase()).not.toContain('<table');
		expect(md.toLowerCase()).not.toContain('<tbody');
		expect(md.toLowerCase()).not.toContain('<tr');
		expect(md).toContain('One');
		expect(md).toContain('Four');
		// First row becomes header; separator inserted so GFM renders a real table
		expect(md).toMatch(/\|[^\n]*---[^\n]*\|/);
	});

	it('keeps Word-style table cells (p in td) on one markdown row per tr', () => {
		const html = `
			<table><tbody>
				<tr><td><p>DESCRIPTION</p></td><td><p>DEFINITION</p></td></tr>
				<tr><td><p>Catastrophic (5)</p></td><td><p>Results to death</p></td></tr>
			</tbody></table>
		`;
		const md = mammothHtmlToSopMarkdown(html);
		expect(md).toContain('DESCRIPTION');
		expect(md).toContain('Results to death');
		const lines = md.split('\n').filter((l) => l.includes('|') && l.trim().startsWith('|'));
		for (const line of lines) {
			expect(line).not.toMatch(/\n/);
		}
		expect(lines.some((l) => l.includes('DESCRIPTION') && l.includes('DEFINITION'))).toBe(true);
	});

	it('still converts a proper header table to markdown pipes', () => {
		const html = `
			<table>
				<thead><tr><th>A</th><th>B</th></tr></thead>
				<tbody><tr><td>1</td><td>2</td></tr></tbody>
			</table>
		`;
		const md = mammothHtmlToSopMarkdown(html);
		expect(md.toLowerCase()).not.toContain('<table');
		expect(md).toContain('|');
		expect(md).toContain('A');
	});

	it('does not leave inline HTML from placeholder paragraphs', () => {
		const html = '<p>Before</p><img src="data:image/png;base64,xx" alt="x"/><p>After</p>';
		const md = mammothHtmlToSopMarkdown(html);
		expect(md.toLowerCase()).not.toContain('<img');
		expect(md.toLowerCase()).not.toContain('<p>');
		expect(md).toMatch(/Image from Word/i);
		expect(md).toMatch(/assets\/filename/);
	});
});
