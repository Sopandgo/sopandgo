import { describe, expect, it } from 'vitest';
import { sanitizeSopHtml } from './sanitizeSopHtml';

describe('sanitizeSopHtml', () => {
	it('strips script and onerror handlers', () => {
		const dirty =
			'<p>hi</p><script>alert(1)</script><img src=x onerror=alert(1)><p onclick="alert(1)">x</p>';
		const clean = sanitizeSopHtml(dirty);
		expect(clean.toLowerCase()).not.toContain('script');
		expect(clean.toLowerCase()).not.toContain('onerror');
		expect(clean.toLowerCase()).not.toContain('onclick');
	});

	it('keeps https links and strips javascript:', () => {
		const dirty = '<a href="https://example.com">ok</a><a href="javascript:alert(1)">bad</a>';
		const clean = sanitizeSopHtml(dirty);
		expect(clean).toContain('https://example.com');
		expect(clean.toLowerCase()).not.toContain('javascript:');
	});

	it('keeps fragment and mailto links', () => {
		const dirty = '<a href="#section">s</a><a href="mailto:a@b.co">m</a>';
		const clean = sanitizeSopHtml(dirty);
		expect(clean).toContain('#section');
		expect(clean).toContain('mailto:a@b.co');
	});

	it('keeps assets/ hrefs', () => {
		const dirty = '<a href="assets/file.pdf">doc</a>';
		expect(sanitizeSopHtml(dirty)).toContain('assets/file.pdf');
	});

	it('keeps asset download img src', () => {
		const id = '550e8400-e29b-41d4-a716-446655440000';
		const dirty = `<img src="/api/assets/download?id=${id}" alt="fig" />`;
		const clean = sanitizeSopHtml(dirty);
		expect(clean).toContain(`/api/assets/download?id=${id}`);
	});

	it('strips disallowed img src', () => {
		const dirty = '<img src="javascript:alert(1)" alt="x" />';
		const clean = sanitizeSopHtml(dirty);
		expect(clean.toLowerCase()).not.toContain('javascript:');
	});

	it('allows figure and figcaption from MarkdownRenderer', () => {
		const dirty =
			'<figure class="not-prose"><img src="https://example.com/x.png" alt="a" /><figcaption>cap</figcaption></figure>';
		const clean = sanitizeSopHtml(dirty);
		expect(clean).toContain('<figure');
		expect(clean).toContain('<figcaption');
	});

	it('allows GFM table structure', () => {
		const dirty = '<table><thead><tr><th>a</th></tr></thead><tbody><tr><td>b</td></tr></tbody></table>';
		const clean = sanitizeSopHtml(dirty);
		expect(clean).toContain('<table');
		expect(clean).toContain('<th>');
	});
});
