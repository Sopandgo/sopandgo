import type { Branding } from '#lib/sdk/types.js';

/** Product name shown when the organization has not set its own. */
export const DEFAULT_BRAND_NAME = 'SOPandGO';

export const DEFAULT_BRANDING: Branding = { name: '', has_logo: false };

const HEX_COLOUR = /^#[0-9a-f]{6}$/i;

/** True when any part of the default look has been replaced. */
export function isCustomBranding(b: Branding | null | undefined): boolean {
	return !!b && (!!b.name || b.has_logo || !!b.primary_light || !!b.primary_dark);
}

export function brandName(b: Branding | null | undefined): string {
	return b?.name || DEFAULT_BRAND_NAME;
}

/** Same-origin logo URL; the hash makes a new upload a new URL. Null without a logo. */
export function brandLogoUrl(b: Branding | null | undefined, variant: 'logo' | 'icon' = 'logo'): string | null {
	if (!b?.has_logo || !b.logo_hash) return null;
	const params = new URLSearchParams({ v: b.logo_hash });
	if (variant === 'icon') params.set('variant', 'icon');
	return `/branding/logo?${params}`;
}

/**
 * CSS that swaps the theme's primary for the organization's colours. The
 * values were checked by the server; they are checked again here because the
 * result goes into a <style> element. Light applies to sop-light and to the
 * system scheme when it is light; dark likewise. Empty when nothing is set.
 */
export function brandingStyle(b: Branding | null | undefined): string {
	const rule = (selector: string, value?: string, content?: string) =>
		value && content && HEX_COLOUR.test(value) && HEX_COLOUR.test(content)
			? `${selector}{--color-primary:${value};--color-primary-content:${content};}`
			: '';
	const system = ':root:not([data-theme])';
	const light = rule('[data-theme="sop-light"]', b?.primary_light, b?.primary_light_content);
	const dark = rule('[data-theme="sop-dark"]', b?.primary_dark, b?.primary_dark_content);
	const systemLight = rule(system, b?.primary_light, b?.primary_light_content);
	const systemDark = rule(system, b?.primary_dark, b?.primary_dark_content);
	return (
		light +
		dark +
		(systemLight ? `@media (prefers-color-scheme: light){${systemLight}}` : '') +
		(systemDark ? `@media (prefers-color-scheme: dark){${systemDark}}` : '')
	);
}

/** Theme backgrounds (base-100 in layout.css); mirrors backend/internal/branding. */
export const THEME_BACKGROUND = { light: '#ffffff', dark: '#2c2a29' } as const;
export const MIN_CONTRAST = 4.5;

export function isHexColour(value: string): boolean {
	return HEX_COLOUR.test(value);
}

/** WCAG 2 contrast ratio of two '#rrggbb' colours; NaN if either is not one. */
export function contrastRatio(a: string, b: string): number {
	if (!HEX_COLOUR.test(a) || !HEX_COLOUR.test(b)) return NaN;
	const luminance = (hex: string) => {
		const channel = (i: number) => {
			const c = parseInt(hex.slice(i, i + 2), 16) / 255;
			return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
		};
		return 0.2126 * channel(1) + 0.7152 * channel(3) + 0.0722 * channel(5);
	};
	const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
	return (hi + 0.05) / (lo + 0.05);
}
