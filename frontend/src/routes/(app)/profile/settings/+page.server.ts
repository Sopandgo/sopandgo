import { fail, redirect } from '@sveltejs/kit';
import * as m from '$lib/paraglide/messages.js';
import { isLocale } from '$lib/paraglide/runtime';
import { isThemePreference } from '$lib/theme';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
    if (!locals.user) {
        throw redirect(303, '/login');
    }
    return { user: locals.user };
};

export const actions: Actions = {
    setLocale: async ({ locals, request }) => {
        const fd = await request.formData();
        const locale = String(fd.get('locale') ?? '').trim();
        if (!isLocale(locale)) {
            return fail(400, { setLocale: { error: 'unsupported' } });
        }
        try {
            await locals.api.auth.updateLocale(locale);
        } catch (err) {
            console.error('Locale update failed:', err);
            return fail(400, { setLocale: { error: 'failed' } });
        }
        throw redirect(303, '/profile/settings');
    },

    setTheme: async ({ locals, request }) => {
        const fd = await request.formData();
        const theme = String(fd.get('theme') ?? '').trim();
        if (!isThemePreference(theme)) {
            return fail(400, { setTheme: { error: 'unsupported' } });
        }
        try {
            await locals.api.auth.updateTheme(theme);
        } catch (err) {
            console.error('Theme update failed:', err);
            return fail(400, { setTheme: { error: 'failed' } });
        }
        throw redirect(303, '/profile/settings');
    },

    changePassword: async ({ locals, request }) => {
        const fd = await request.formData();
        const current_password = String(fd.get('current_password') ?? '');
        const new_password = String(fd.get('new_password') ?? '');
        const confirm_password = String(fd.get('confirm_password') ?? '');

        if (new_password !== confirm_password) {
            return fail(400, {
                changePassword: { error: m.error_passwords_mismatch() }
            });
        }

        const user_id = locals.user?.id;
        if (!user_id) {
            return fail(401, {
                changePassword: { error: m.error_not_authenticated() }
            });
        }

        try {
            await locals.api.auth.updatePassword(current_password, new_password);
            await locals.api.auth.logout();
        } catch (err: unknown) {
            console.error('Change Password Error:', err);
            const message = err instanceof Error ? err.message : '';
            const msg =
                message === 'WEAK_PASSWORD'
                    ? m.error_weak_password()
                    : message === 'WRONG_CURRENT_PASSWORD'
                      ? m.error_current_password_wrong()
                      : message === 'TOO_MANY_ATTEMPTS'
                        ? m.error_too_many_attempts()
                        : m.error_password_update_failed();

            return fail(400, {
                changePassword: { error: msg }
            });
        }

        throw redirect(303, '/login?passwordChanged=true');
    },

    signOutOthers: async ({ locals }) => {
        try {
            await locals.api.auth.signOutOtherSessions();
        } catch (err) {
            console.error('Sign out other sessions failed:', err);
            return fail(400, { signOutOthers: { ok: false, error: 'failed' } });
        }
        return { signOutOthers: { ok: true, error: null } };
    }
};
