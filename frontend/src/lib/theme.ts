export const themePreferences = ['light', 'dark', 'system'] as const;

export type ThemePreference = (typeof themePreferences)[number];

export function isThemePreference(value: string): value is ThemePreference {
	return (themePreferences as readonly string[]).includes(value);
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
