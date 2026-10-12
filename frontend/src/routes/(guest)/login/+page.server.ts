import { fail, redirect, type Cookies } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { TwoFactorMethod } from '#lib/sdk/types.js';
import type { Actions, PageServerLoad } from './$types';

/**
 * Between the password and the second factor, the challenge token and the
 * methods it allows live in an httpOnly cookie scoped to /login. The token
 * proves only the password step and expires with the backend's challenge.
 */
const CHALLENGE_COOKIE = 'two_factor_challenge';

interface PendingChallenge {
    token: string;
    methods: TwoFactorMethod[];
}

function readChallenge(cookies: Cookies): PendingChallenge | null {
    const raw = cookies.get(CHALLENGE_COOKIE);
    if (!raw) return null;
    try {
        const parsed = JSON.parse(raw) as PendingChallenge;
        if (typeof parsed.token === 'string' && Array.isArray(parsed.methods)) return parsed;
    } catch {
        /* fall through */
    }
    return null;
}

function clearChallenge(cookies: Cookies) {
    cookies.delete(CHALLENGE_COOKIE, { path: '/login' });
}

const METHODS: TwoFactorMethod[] = ['totp', 'email', 'recovery_code'];

export const load: PageServerLoad = async ({ cookies }) => {
    const pending = readChallenge(cookies);
    return { twoFactor: pending ? { methods: pending.methods } : null };
};

export const actions: Actions = {
    password: async ({ request, locals, cookies, url }) => {
        const data = await request.formData();
        const email = data.get('email')?.toString();
        const password = data.get('password')?.toString();

        if (!email || !password) {
            return fail(400, { message: m.error_missing_fields() });
        }

        try {
            // SDK handles API call + Cookie setting automatically
            const result = await locals.api.auth.login(email, password);
            if (result.kind === 'two_factor') {
                cookies.set(CHALLENGE_COOKIE, JSON.stringify({ token: result.challenge_token, methods: result.methods }), {
                    path: '/login',
                    httpOnly: true,
                    sameSite: 'strict',
                    secure: url.protocol === 'https:',
                    maxAge: result.expires_in
                });
                return { twoFactor: { methods: result.methods } };
            }
        } catch (err) {
            const e = err as Error;

            if (e.message === 'INVALID_CREDENTIALS') {
                return fail(401, { message: m.error_invalid_credentials(), email });
            }
            if (e.message === 'ACCOUNT_INACTIVE') {
                return fail(403, { message: m.error_account_disabled(), email });
            }
            if (e.message === 'TOO_MANY_ATTEMPTS') {
                return fail(429, { message: m.error_too_many_attempts(), email });
            }

            console.error('Login Error:', e);
            return fail(500, { message: m.error_system(), email });
        }

        throw redirect(303, '/dashboard');
    },

    verify: async ({ request, locals, cookies }) => {
        const pending = readChallenge(cookies);
        if (!pending) {
            return fail(401, { message: m.two_factor_login_expired() });
        }
        const data = await request.formData();
        const code = String(data.get('code') ?? '').trim();
        const method = String(data.get('method') ?? 'totp') as TwoFactorMethod;
        if (!METHODS.includes(method) || !pending.methods.includes(method)) {
            return fail(400, { message: m.error_system(), twoFactor: { methods: pending.methods }, method: 'totp' });
        }
        if (!code) {
            return fail(400, { message: m.two_factor_code_required(), twoFactor: { methods: pending.methods }, method });
        }

        try {
            await locals.api.auth.verifyTwoFactor(pending.token, method, code);
        } catch (err) {
            const e = err as Error;
            switch (e.message) {
                case 'CHALLENGE_EXPIRED':
                    clearChallenge(cookies);
                    return fail(401, { message: m.two_factor_login_expired() });
                case 'ACCOUNT_INACTIVE':
                    clearChallenge(cookies);
                    return fail(403, { message: m.error_account_disabled() });
                case 'INVALID_CODE':
                    return fail(401, { message: m.two_factor_code_invalid(), twoFactor: { methods: pending.methods }, method });
                case 'TOO_MANY_ATTEMPTS':
                    return fail(429, { message: m.error_too_many_attempts(), twoFactor: { methods: pending.methods }, method });
                case 'AUTHENTICATOR_UNAVAILABLE':
                    return fail(409, { message: m.two_factor_authenticator_unavailable(), twoFactor: { methods: pending.methods }, method });
                default:
                    console.error('Two-factor sign-in error:', e);
                    return fail(500, { message: m.error_system(), twoFactor: { methods: pending.methods }, method });
            }
        }

        clearChallenge(cookies);
        throw redirect(303, '/dashboard');
    },

    sendEmail: async ({ locals, cookies }) => {
        const pending = readChallenge(cookies);
        if (!pending) {
            return fail(401, { message: m.two_factor_login_expired() });
        }
        try {
            await locals.api.auth.sendTwoFactorEmail(pending.token);
        } catch (err) {
            const e = err as Error;
            const keep = { twoFactor: { methods: pending.methods }, method: 'email' as const };
            switch (e.message) {
                case 'CHALLENGE_EXPIRED':
                    clearChallenge(cookies);
                    return fail(401, { message: m.two_factor_login_expired() });
                case 'EMAIL_THROTTLED':
                case 'TOO_MANY_ATTEMPTS':
                    return fail(429, { message: m.two_factor_email_throttled(), ...keep });
                default:
                    console.error('Two-factor email error:', e);
                    return fail(502, { message: m.two_factor_email_failed(), ...keep });
            }
        }
        return { twoFactor: { methods: pending.methods }, method: 'email' as const, emailSent: true };
    },

    cancel: async ({ cookies }) => {
        clearChallenge(cookies);
        return { twoFactor: null };
    }
};
