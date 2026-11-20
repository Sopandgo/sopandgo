import { describe, expect, it } from 'vitest';
import {
    DEMO_INSTALL_URL,
    demoCookieOptions,
    demoSessionToOpen,
    isDemoMode,
    isDemoReadOnlyError,
    otherDemoRole
} from './demo';

describe('isDemoMode', () => {
    it('is on only for true, in any case, ignoring spaces', () => {
        expect(isDemoMode('true')).toBe(true);
        expect(isDemoMode(' TRUE ')).toBe(true);
        for (const value of [undefined, '', 'false', '1', 'yes']) {
            expect(isDemoMode(value)).toBe(false);
        }
    });
});

describe('demoSessionToOpen', () => {
    it('opens a researcher session for a visitor without a session', () => {
        expect(demoSessionToOpen(true, false)).toBe('viewer');
    });

    it('keeps the session a returning visitor already has, whatever its role', () => {
        expect(demoSessionToOpen(true, true)).toBeNull();
    });

    it('never opens a session outside demo mode', () => {
        expect(demoSessionToOpen(false, false)).toBeNull();
        expect(demoSessionToOpen(false, true)).toBeNull();
    });
});

describe('otherDemoRole', () => {
    it('switches between researcher and auditor', () => {
        expect(otherDemoRole('viewer')).toBe('auditor');
        expect(otherDemoRole('auditor')).toBe('viewer');
    });

    it('falls back to the researcher for any other role', () => {
        expect(otherDemoRole('admin')).toBe('viewer');
    });
});

describe('demoCookieOptions', () => {
    it('keeps the refresh cookie for the four-hour demo session', () => {
        const { access, refresh } = demoCookieOptions(true);
        expect(refresh.maxAge).toBe(4 * 60 * 60);
        expect(access.maxAge).toBe(5 * 60);
        expect(refresh).toMatchObject({ path: '/', httpOnly: true, secure: true, sameSite: 'lax' });
    });

    it('drops the secure flag on plain http', () => {
        expect(demoCookieOptions(false).refresh.secure).toBe(false);
    });
});

describe('isDemoReadOnlyError', () => {
    it('recognizes the API refusal the SDK throws', () => {
        expect(isDemoReadOnlyError(new Error('DEMO_READ_ONLY'))).toBe(true);
        expect(isDemoReadOnlyError(new Error('LOGIN_FAILED'))).toBe(false);
        expect(isDemoReadOnlyError('DEMO_READ_ONLY')).toBe(false);
    });
});

describe('DEMO_INSTALL_URL', () => {
    it('links to the README on GitHub', () => {
        expect(DEMO_INSTALL_URL).toBe('https://github.com/Sopandgo/sopandgo#readme');
    });
});
