import DOMPurify from 'isomorphic-dompurify';

let hooksInstalled = false;

function installHooks(): void {
	if (hooksInstalled) {
		return;
	}
	hooksInstalled = true;

	DOMPurify.addHook('uponSanitizeAttribute', (node, data) => {
		const el = node as Element;
		const tag = el.tagName?.toLowerCase();
		const name = data.attrName?.toLowerCase();

		if (tag === 'img' && name === 'src') {
			const v = data.attrValue.trim();
			if (isAllowedImgSrc(v)) {
				return;
			}
			data.keepAttr = false;
			return;
		}

		if (tag === 'a' && name === 'href') {
			const v = data.attrValue.trim();
			if (isAllowedAHref(v)) {
				return;
			}
			data.keepAttr = false;
		}
	});
}

const uuidParam = /^\/api\/assets\/download\?id=[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function isAllowedImgSrc(src: string): boolean {
	if (src.startsWith('https://') || src.startsWith('http://')) {
		return true;
	}
	return uuidParam.test(src);
}

function isAllowedAHref(href: string): boolean {
	const h = href.trim();
	if (h === '' || h.startsWith('#')) {
		return true;
	}
	const lower = h.toLowerCase();
	if (lower.startsWith('mailto:')) {
		return true;
	}
	if (lower.startsWith('https://') || lower.startsWith('http://')) {
		return true;
	}
	if (lower.startsWith('assets/')) {
		return true;
	}
	return false;
}

/**
 * Sanitizes HTML produced from SOP markdown before {@html} rendering.
 * Aligns loosely with backend markdown URL policy + safe DOM output.
 */
export function sanitizeSopHtml(dirty: string): string {
	if (!dirty) {
		return '';
	}
	installHooks();
	return DOMPurify.sanitize(dirty, {
		ADD_TAGS: ['figure', 'figcaption', 'input'],
		ADD_ATTR: ['class', 'id', 'title', 'alt', 'colspan', 'rowspan', 'align', 'start'],
		ALLOW_DATA_ATTR: false
	});
}
