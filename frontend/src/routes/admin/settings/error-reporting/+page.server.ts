import { fail } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
    let settings = null;
    try {
        settings = await locals.api.admin.getErrorReporting();
    } catch (err) {
        console.error('Failed to load error reporting settings:', err);
    }
    return { settings };
};

function saveError(code: string): string {
    switch (code) {
        case 'dsn_required':
            return m.error_reporting_dsn_required();
        case 'dsn_invalid':
            return m.error_reporting_dsn_invalid();
        case 'environment_invalid':
            return m.error_reporting_environment_invalid();
        default:
            return m.error_reporting_save_failed();
    }
}

export const actions: Actions = {
    save: async ({ locals, request }) => {
        const fd = await request.formData();
        const values = {
            enabled: fd.get('enabled') === 'on',
            dsn: String(fd.get('dsn') ?? '').trim(),
            environment: String(fd.get('environment') ?? '').trim(),
            browser_enabled: fd.get('browser_enabled') === 'on'
        };
        try {
            await locals.api.admin.updateErrorReporting(values);
            return { save: { ok: true } };
        } catch (err) {
            const code = err instanceof Error ? err.message : '';
            if (!['dsn_required', 'dsn_invalid', 'environment_invalid'].includes(code)) {
                console.error('Error reporting save failed:', err);
            }
            // Hand the values back so a rejected form keeps what the admin typed.
            return fail(code.startsWith('ERROR_REPORTING') ? 500 : 400, {
                save: { error: saveError(code), values }
            });
        }
    },

    test: async ({ locals }) => {
        try {
            const eventId = await locals.api.admin.sendErrorReportingTest();
            return { test: { ok: true, eventId } };
        } catch (err) {
            const code = err instanceof Error ? err.message : '';
            if (code === 'not_enabled') {
                return fail(409, { test: { error: m.error_reporting_test_not_enabled() } });
            }
            if (code !== 'delivery_failed') console.error('Error reporting test failed:', err);
            return fail(502, { test: { error: m.error_reporting_test_failed() } });
        }
    }
};
