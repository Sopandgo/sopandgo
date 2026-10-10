import { fail } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
    let smtp = null;
    try {
        smtp = await locals.api.admin.getEmailSettings();
    } catch (err) {
        console.error('Failed to load email settings:', err);
    }
    return { smtp };
};

export const actions: Actions = {
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
