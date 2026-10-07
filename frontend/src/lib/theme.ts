export const themePreferences = ['light', 'dark', 'system'] as const;

export type ThemePreference = (typeof themePreferences)[number];

/** Non-httpOnly cookie for appearance before sign-in. Signed-in users ignore it. */
export const themeCookieName = 'theme';

/** Match the locale cookie lifetime (400 days). */
export const themeCookieMaxAge = 60 * 60 * 24 * 400;

export function isThemePreference(value: string): value is ThemePreference {
	return (themePreferences as readonly string[]).includes(value);
}

/** Guest appearance. Only an explicit light or dark cookie overrides the system scheme. */
export function guestThemeFromCookie(value: string | undefined): ThemePreference {
	if (value === 'light' || value === 'dark') return value;
	return 'system';
}

/** `document.cookie` assignment for a guest light/dark choice. */
export function themeCookieAssignment(theme: 'light' | 'dark', secure: boolean): string {
	const secureAttr = secure ? '; Secure' : '';
	return `${themeCookieName}=${theme}; Path=/; Max-Age=${themeCookieMaxAge}; SameSite=Lax${secureAttr}`;
}

/** DaisyUI theme name. Null follows the system color scheme. */
export function daisyTheme(theme: string): 'corporate' | 'business' | null {
	if (theme === 'light') return 'corporate';
	if (theme === 'dark') return 'business';
	return null;
}

/** Attribute fragment for `<html>`, including a leading space when set. */
export function themeAttribute(theme: string): string {
	const name = daisyTheme(theme);
	return name ? ` data-theme="${name}"` : '';
}
