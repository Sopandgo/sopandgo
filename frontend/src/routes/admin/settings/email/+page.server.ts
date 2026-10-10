import { fail } from '@sveltejs/kit';
import * as m from '$lib/paraglide/messages.js';
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
    // The SMTP and Resend header toggles. One transport sends at a time: switching one
    // on makes it the transport and turns email on; switching the sending one off
    // falls back to manual links.
    setDelivery: async ({ locals, request }) => {
        const fd = await request.formData();
        const transport = String(fd.get('transport') ?? '').trim();
        const enabled = fd.get('enabled') === 'on';
        if (transport !== 'smtp' && transport !== 'resend') {
            return fail(400, { setDelivery: { error: m.error_select_transport() } });
        }
        try {
            if (enabled) {
                await locals.api.admin.updateMailTransport(transport);
                await locals.api.admin.updateMailMode('smtp');
            } else {
                const current = await locals.api.admin.getEmailSettings();
                if (current.mail_transport === transport) {
                    await locals.api.admin.updateMailMode('manual_links');
                }
            }
            return { setDelivery: { ok: true, transport } };
        } catch (err) {
            console.error('Mail delivery update failed:', err);
            return fail(500, { setDelivery: { error: m.error_mail_delivery_toggle(), transport } });
        }
    },

    saveResend: async ({ locals, request }) => {
        const fd = await request.formData();
        const from_address = String(fd.get('resend_from_address') ?? '').trim();
        const api_key = String(fd.get('resend_api_key') ?? '');

        if (!from_address) {
            return fail(400, {
                saveResend: { error: m.error_from_required() },
                resendValues: { from_address }
            });
        }

        try {
            await locals.api.admin.updateResendSettings({ from_address, api_key });
            return { saveResend: { ok: true } };
        } catch (err: unknown) {
            const code = err instanceof Error ? err.message : '';
            if (code === 'SMTP_ENCRYPTION_KEY_MISSING') {
                return fail(400, {
                    saveResend: {
                        error: m.error_encryption_key_secrets()
                    },
                    resendValues: { from_address }
                });
            }
            console.error('Resend save failed:', err);
            return fail(500, {
                saveResend: { error: m.error_resend_save() },
                resendValues: { from_address }
            });
        }
    },


    save: async ({ locals, request }) => {
        const fd = await request.formData();
        const host = String(fd.get('host') ?? '').trim();
        const port = String(fd.get('port') ?? '').trim();
        const username = String(fd.get('username') ?? '').trim();
        const from_address = String(fd.get('from_address') ?? '').trim();
        const password = String(fd.get('password') ?? '');

        if (!host || !port || !username || !from_address) {
            return fail(400, {
                save: { error: m.error_smtp_fields() },
                values: { host, port, username, from_address }
            });
        }

        try {
            await locals.api.admin.updateSmtpSettings({
                host,
                port,
                username,
                from_address,
                password
            });
            return { save: { ok: true } };
        } catch (err: unknown) {
            const code = err instanceof Error ? err.message : '';
            if (code === 'SMTP_ENCRYPTION_KEY_MISSING') {
                return fail(400, {
                    save: {
                        error: m.error_encryption_key_smtp()
                    },
                    values: { host, port, username, from_address }
                });
            }
            console.error('SMTP save failed:', err);
            return fail(500, {
                save: { error: m.error_smtp_save() },
                values: { host, port, username, from_address }
            });
        }
    },

    test: async ({ locals, request }) => {
        const fd = await request.formData();
        const test_to = String(fd.get('test_to') ?? '').trim().toLowerCase();

        if (!test_to || !test_to.includes('@')) {
            return fail(400, { test: { error: m.error_test_email() } });
        }

        try {
            await locals.api.admin.sendSmtpTest(test_to);
            return { test: { ok: true } };
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : 'SMTP_TEST_FAILED';
            if (msg === 'MAIL_TEST_NOT_READY' || msg === 'SMTP_NOT_READY') {
                return fail(400, {
                    test: {
                        error: m.error_mail_not_ready()
                    }
                });
            }
            console.error('SMTP test failed:', err);
            return fail(500, {
                test: { error: msg === 'SMTP_TEST_FAILED' ? m.error_test_email_failed() : msg }
            });
        }
    }
};
