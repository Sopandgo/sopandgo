import { describe, expect, it } from 'vitest';
import {
	brandLogoUrl,
	brandName,
	brandingStyle,
	contrastRatio,
	DEFAULT_BRAND_NAME,
	isCustomBranding,
	MIN_CONTRAST,
	THEME_BACKGROUND
} from './branding';

describe('branding', () => {
	it('falls back to the product name and gopher', () => {
		expect(brandName({ name: '', has_logo: false })).toBe(DEFAULT_BRAND_NAME);
		expect(brandName(null)).toBe(DEFAULT_BRAND_NAME);
		expect(brandLogoUrl({ name: '', has_logo: false })).toBeNull();
		expect(isCustomBranding({ name: '', has_logo: false })).toBe(false);
		expect(brandingStyle({ name: '', has_logo: false })).toBe('');
	});

	it('builds cache-busting logo URLs', () => {
		const b = { name: 'Lab', has_logo: true, logo_hash: 'abc' };
		expect(brandLogoUrl(b)).toBe('/branding/logo?v=abc');
		expect(brandLogoUrl(b, 'icon')).toBe('/branding/logo?v=abc&variant=icon');
		expect(isCustomBranding(b)).toBe(true);
	});

	it('overrides primary per theme, including the system scheme', () => {
		const css = brandingStyle({
			name: '',
			has_logo: false,
			primary_light: '#1d4ed8',
			primary_light_content: '#ffffff',
			primary_dark: '#93c5fd',
			primary_dark_content: '#2c2a29'
		});
		expect(css).toContain('[data-theme="sop-light"]{--color-primary:#1d4ed8;--color-primary-content:#ffffff;}');
		expect(css).toContain('[data-theme="sop-dark"]{--color-primary:#93c5fd;');
		expect(css).toContain('@media (prefers-color-scheme: light){:root:not([data-theme]){--color-primary:#1d4ed8;');
		expect(css).toContain('@media (prefers-color-scheme: dark){:root:not([data-theme]){--color-primary:#93c5fd;');
	});

	it('leaves the other theme alone when only one colour is set', () => {
		const css = brandingStyle({
			name: '',
			has_logo: false,
			primary_light: '#1d4ed8',
			primary_light_content: '#ffffff'
		});
		expect(css).not.toContain('sop-dark');
		expect(css).not.toContain('prefers-color-scheme: dark');
	});

	it('drops anything that is not a hex colour', () => {
		const css = brandingStyle({
			name: '',
			has_logo: false,
			primary_light: '#fff;}</style><script>',
			primary_light_content: '#ffffff'
		});
		expect(css).toBe('');
	});

	it('computes WCAG contrast like the backend', () => {
		expect(contrastRatio('#000000', '#ffffff')).toBeCloseTo(21, 5);
		expect(contrastRatio('#767676', THEME_BACKGROUND.light)).toBeGreaterThanOrEqual(MIN_CONTRAST);
		expect(contrastRatio('#777777', THEME_BACKGROUND.light)).toBeLessThan(MIN_CONTRAST);
		expect(contrastRatio('blue', '#ffffff')).toBeNaN();
	});
});
