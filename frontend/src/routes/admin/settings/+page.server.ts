import { fail } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { TwoFactorPolicy } from '#lib/sdk/types.js';
import type { Actions, PageServerLoad } from './$types';

const POLICIES: TwoFactorPolicy[] = ['none', 'email', 'totp'];

export const load: PageServerLoad = async ({ locals }) => {
    let smtp = null;
    try {
        smtp = await locals.api.admin.getEmailSettings();
    } catch (err) {
        console.error('Failed to load email settings:', err);
    }
    let security = null;
    try {
        security = await locals.api.admin.getSecuritySettings();
    } catch (err) {
        console.error('Failed to load security settings:', err);
    }
    return { smtp, security };
};

export const actions: Actions = {
    saveSecurity: async ({ locals, request }) => {
        const fd = await request.formData();
        const policy = String(fd.get('two_factor_policy') ?? '') as TwoFactorPolicy;
        if (!POLICIES.includes(policy)) {
            return fail(400, { saveSecurity: { error: m.security_save_failed() } });
        }
        try {
            await locals.api.admin.updateSecuritySettings(policy);
            return { saveSecurity: { ok: true } };
        } catch (err) {
            const code = err instanceof Error ? err.message : '';
            if (code === 'ACTOR_NEEDS_TWO_FACTOR') {
                return fail(409, { saveSecurity: { error: m.security_policy_needs_own() } });
            }
            if (code === 'EMAIL_TRANSPORT_OFF') {
                return fail(409, { saveSecurity: { error: m.security_policy_needs_email() } });
            }
            if (code === 'ENCRYPTION_KEY_MISSING') {
                return fail(409, { saveSecurity: { error: m.two_factor_key_missing() } });
            }
            console.error('Security settings update failed:', err);
            return fail(500, { saveSecurity: { error: m.security_save_failed() } });
        }
    },

    saveDefaultLocale: async ({ locals, request }) => {
        const fd = await request.formData();
        const default_locale = String(fd.get('locale') ?? '').trim();
        if (!default_locale) {
            return fail(400, { saveDefaultLocale: { error: m.error_choose_language() } });
        }
        try {
            await locals.api.admin.updateDefaultLocale(default_locale);
            return { saveDefaultLocale: { ok: true, locale: default_locale } };
        } catch (err) {
            console.error('Default locale update failed:', err);
            const unsupported = err instanceof Error && err.message === 'UNSUPPORTED_LOCALE';
            return fail(unsupported ? 400 : 500, {
                saveDefaultLocale: {
                    error: unsupported ? m.error_language_unavailable() : m.error_org_language()
                }
            });
        }
    }
};
