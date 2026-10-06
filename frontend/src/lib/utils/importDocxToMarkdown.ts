import mammoth from 'mammoth';
import TurndownService from 'turndown';
import { gfm } from 'turndown-plugin-gfm';
import * as m from '$lib/paraglide/messages.js';

/** Shown in the editor where Word had an embedded image (no upload). */
function imagePlaceholderHtml(): string {
	return `<p><strong>${m.editor_word_image()}</strong> ${m.editor_word_image_help()} <code>![description](assets/filename)</code>.</p>`;
}

function replaceImagesWithPlaceholders(html: string): string {
	let out = html.replace(/<img\b[^>]*>/gi, imagePlaceholderHtml());
	out = out.replace(/<figure>\s*<\/figure>/gi, '');
	out = out.replace(/<p>\s*<\/p>/gi, '');
	return out;
}

function cellDepth(el: Element): number {
	let d = 0;
	for (let p = el.parentElement; p; p = p.parentElement) d++;
	return d;
}

/**
 * Mammoth puts &lt;p&gt;, &lt;br&gt;, lists, etc. inside &lt;td&gt;/&lt;th&gt;. Turndown turns
 * each &lt;p&gt; into block breaks, so pipe rows split across many lines (broken GFM tables).
 * Collapse each cell to one line of plain text before Turndown.
 */
function flattenTableCells(html: string): string {
	if (typeof DOMParser !== 'undefined') {
		try {
			const doc = new DOMParser().parseFromString(
				`<div id="sop-import-root">${html}</div>`,
				'text/html'
			);
			const root = doc.getElementById('sop-import-root');
			if (root) {
				const cells = Array.from(root.querySelectorAll('td, th'));
				cells.sort((a, b) => cellDepth(b) - cellDepth(a));
				for (const cell of cells) {
					const t = (cell.textContent || '').replace(/\s+/g, ' ').trim();
					cell.textContent = t;
				}
				return root.innerHTML;
			}
		} catch {
			/* fall through */
		}
	}
	return flattenTableCellsRegex(html);
}

function decodeBasicEntities(s: string): string {
	const safeCodePoint = (n: number) => {
		try {
			if (n < 0 || n > 0x10ffff) return '\ufffd';
			return String.fromCodePoint(n);
		} catch {
			return '\ufffd';
		}
	};
	return s
		.replace(/&nbsp;/gi, ' ')
		.replace(/&#(\d+);/g, (_, n) => safeCodePoint(Number.parseInt(n, 10)))
		.replace(/&#x([0-9a-f]+);/gi, (_, h) => safeCodePoint(Number.parseInt(h, 16)))
		.replace(/&amp;/g, '&')
		.replace(/&lt;/g, '<')
		.replace(/&gt;/g, '>')
		.replace(/&quot;/g, '"');
}

function cellInnerToPlain(inner: string): string {
	const decoded = decodeBasicEntities(inner);
	return decoded
		.replace(/<br\s*\/?>/gi, ' ')
		.replace(/<\/p>\s*<p\b[^>]*>/gi, ' ')
		.replace(/<\/div>\s*<div\b[^>]*>/gi, ' ')
		.replace(/<[^>]+>/g, ' ')
		.replace(/\s+/g, ' ')
		.trim();
}

function escapeHtmlText(text: string): string {
	return text
		.replace(/&/g, '&amp;')
		.replace(/</g, '&lt;')
		.replace(/>/g, '&gt;')
		.replace(/"/g, '&quot;');
}

/** Innermost &lt;td&gt;/&lt;th&gt; first (no nested table cells), for Node / tests without DOMParser. */
function flattenTableCellsRegex(html: string): string {
	let cur = html;
	let prev = '';
	while (cur !== prev) {
		prev = cur;
		cur = cur.replace(/<(td|th)(\s[^>]*)?>([\s\S]*?)<\/\1>/gi, (full, tag: string, attrs: string, inner: string) => {
			if (/<(td|th)\b/i.test(inner)) {
				return full;
			}
			const plain = cellInnerToPlain(inner);
			return `<${tag}${attrs ?? ''}>${escapeHtmlText(plain)}</${tag}>`;
		});
	}
	return cur;
}

function createTurndownForSop(): TurndownService {
	const turndown = new TurndownService({
		headingStyle: 'atx',
		codeBlockStyle: 'fenced',
		bulletListMarker: '-'
	});
	turndown.use(gfm);
	// turndown-plugin-gfm uses `keep()` for tables without a GFM header row, which embeds raw
	// HTML in the markdown. Our API forbids raw HTML — register a rule that runs first (addRule
	// prepends) so every table is converted to pipe-style markdown instead of kept as HTML.
	turndown.addRule('sopAllTablesToMarkdown', {
		filter: 'table',
		replacement(content) {
			const c = content.replace('\n\n', '\n');
			return '\n\n' + c + '\n\n';
		}
	});
	return turndown;
}

function isPipeTableRow(line: string): boolean {
	const t = line.trim();
	if (!t.startsWith('|') || !t.endsWith('|') || t.length < 3) return false;
	return (t.match(/\|/g) ?? []).length >= 2;
}

function isGfmAlignmentRow(line: string): boolean {
	const t = line.trim();
	if (!t.startsWith('|') || !t.endsWith('|')) return false;
	const inner = t.slice(1, -1).split('|');
	return (
		inner.length > 0 &&
		inner.every((cell) => {
			const c = cell.trim();
			return /^:?-{3,}:?$/.test(c);
		})
	);
}

function separatorRowForHeaderRow(headerRow: string): string | null {
	const t = headerRow.trim();
	const inner = t.slice(1, -1);
	const cells = inner.split('|');
	if (cells.length === 0) return null;
	return '|' + cells.map(() => ' --- ').join('|') + '|';
}

/**
 * Word tables without &lt;th&gt; become pipe rows without a GFM `| --- |` line, so many
 * renderers show plain text instead of a grid. Insert that row when missing (only affects
 * contiguous `| ... |` blocks, not fenced code — rare in Word imports).
 */
function insertMissingGfmTableSeparators(markdown: string): string {
	const lines = markdown.split(/\r?\n/);
	const out: string[] = [];
	let i = 0;
	while (i < lines.length) {
		const line = lines[i];
		if (!isPipeTableRow(line)) {
			out.push(line);
			i++;
			continue;
		}
		const block: string[] = [];
		while (i < lines.length && isPipeTableRow(lines[i])) {
			block.push(lines[i]);
			i++;
		}
		const secondIsSep = block.length >= 2 && isGfmAlignmentRow(block[1]);
		if (!secondIsSep) {
			const sep = separatorRowForHeaderRow(block[0]);
			if (sep) {
				out.push(block[0], sep, ...block.slice(1));
			} else {
				out.push(...block);
			}
		} else {
			out.push(...block);
		}
	}
	return out.join('\n');
}

/**
 * Converts HTML from Mammoth into markdown suitable for the SOP editor and backend validation.
 * Exported for unit tests.
 */
export function mammothHtmlToSopMarkdown(html: string): string {
	const withPlaceholders = replaceImagesWithPlaceholders(html);
	const tableSafe = flattenTableCells(withPlaceholders);
	const turndown = createTurndownForSop();
	const raw = turndown.turndown(tableSafe).trim();
	return insertMissingGfmTableSeparators(raw);
}

export interface ImportDocxResult {
	markdown: string;
	messages: string[];
}

/**
 * Converts a .docx file to markdown suitable for the SOP editor.
 * Embedded images are not extracted; each becomes a visible placeholder paragraph.
 */
export async function importDocxToMarkdown(file: File): Promise<ImportDocxResult> {
	const lower = file.name.toLowerCase();
	if (!lower.endsWith('.docx')) {
		throw new Error(m.error_import_docx_type());
	}

	const arrayBuffer = await file.arrayBuffer();
	const { value: html, messages } = await mammoth.convertToHtml({ arrayBuffer });
	const markdown = mammothHtmlToSopMarkdown(html);

	if (!markdown) {
		throw new Error(m.error_import_docx_empty());
	}

	return {
		markdown,
		messages: messages.map((note) => note.message)
	};
}
