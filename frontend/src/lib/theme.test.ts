import { describe, expect, it } from 'vitest';
import { daisyTheme, isThemePreference, themeAttribute } from './theme';

describe('theme preference', () => {
	it('maps light and dark onto the DaisyUI themes and leaves system unset', () => {
		expect(daisyTheme('light')).toBe('corporate');
		expect(daisyTheme('dark')).toBe('business');
		expect(daisyTheme('system')).toBeNull();
		expect(themeAttribute('dark')).toBe(' data-theme="business"');
		expect(themeAttribute('system')).toBe('');
		expect(isThemePreference('sepia')).toBe(false);
	});
});
