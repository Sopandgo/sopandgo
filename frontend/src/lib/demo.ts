/*
 * Session decisions for a read-only public demo (DEMO_MODE). The SvelteKit
 * server reads the flag; the Go API refuses writes on its own. hooks.server.ts
 * calls these and stays thin.
 */

/** The two roles a demo visitor can view the app as. */
export type DemoRole = 'viewer' | 'auditor';

/** Matches the API's demo session lifetime: refresh tokens last four hours. */
export const DEMO_SESSION_SECONDS = 4 * 60 * 60;

/** Access tokens last five minutes, as for a normal sign-in. */
const ACCESS_TOKEN_SECONDS = 5 * 60;

/** Where the banner and every disabled control send visitors who want to edit. */
export const DEMO_INSTALL_URL = 'https://github.com/Sopandgo/sopandgo#readme';

/** The error the SDK throws when the API answers 403 `demo_read_only`. */
export const DEMO_READ_ONLY = 'DEMO_READ_ONLY';

/** DEMO_MODE is on only for "true", in any case, ignoring surrounding space. */
export function isDemoMode(value: string | undefined): boolean {
    return (value ?? '').trim().toLowerCase() === 'true';
}

/**
 * Which demo session to open for this request: the researcher for a visitor
 * without a session, nothing for a visitor who has one (so a returning visitor
 * keeps their role), and never anything outside demo mode.
 */
export function demoSessionToOpen(demo: boolean, signedIn: boolean): DemoRole | null {
    if (!demo || signedIn) return null;
    return 'viewer';
}

/** The role the banner's switch moves to from the current one. */
export function otherDemoRole(role: string): DemoRole {
    return role === 'viewer' ? 'auditor' : 'viewer';
}

/** Cookie options for a demo session; the refresh cookie lives as long as the session. */
export function demoCookieOptions(secure: boolean) {
    const base = { path: '/', httpOnly: true, secure, sameSite: 'lax' as const };
    return {
        access: { ...base, maxAge: ACCESS_TOKEN_SECONDS },
        refresh: { ...base, maxAge: DEMO_SESSION_SECONDS }
    };
}

/** True for the error the SDK throws when the demo API refuses a write. */
export function isDemoReadOnlyError(err: unknown): boolean {
    return err instanceof Error && err.message === DEMO_READ_ONLY;
}
