import { describe, expect, it } from 'vitest';
import {
	daisyTheme,
	guestThemeFromCookie,
	isThemePreference,
	themeAttribute,
	themeCookieAssignment,
	themeCookieName
} from './theme';

describe('theme preference', () => {
	it('maps light and dark onto the DaisyUI themes and leaves system unset', () => {
		expect(daisyTheme('light')).toBe('corporate');
		expect(daisyTheme('dark')).toBe('business');
		expect(daisyTheme('system')).toBeNull();
		expect(themeAttribute('dark')).toBe(' data-theme="business"');
		expect(themeAttribute('system')).toBe('');
		expect(isThemePreference('sepia')).toBe(false);
	});

	it('reads only an explicit light or dark guest cookie', () => {
		expect(guestThemeFromCookie(undefined)).toBe('system');
		expect(guestThemeFromCookie('light')).toBe('light');
		expect(guestThemeFromCookie('dark')).toBe('dark');
		expect(guestThemeFromCookie('system')).toBe('system');
		expect(guestThemeFromCookie('sepia')).toBe('system');
	});

	it('writes a non-httpOnly theme cookie', () => {
		expect(themeCookieAssignment('dark', false)).toBe(
			`${themeCookieName}=dark; Path=/; Max-Age=34560000; SameSite=Lax`
		);
		expect(themeCookieAssignment('light', true)).toContain('; Secure');
	});
});
