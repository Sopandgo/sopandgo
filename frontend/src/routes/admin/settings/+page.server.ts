import { fail } from '@sveltejs/kit';
import * as m from '$lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';
import type { IntegrationEvent } from '$lib/sdk/types';

const ALL_EVENTS: IntegrationEvent[] = [
    'sop_published',
    'sop_rc',
    'sop_rejected',
    'backup_s3_failed',
    'integrity_check_failed'
];

function parseEvents(fd: FormData): IntegrationEvent[] {
    const selected = fd.getAll('events').map((v) => String(v));
    return ALL_EVENTS.filter((e) => selected.includes(e));
}

export const load: PageServerLoad = async ({ locals }) => {
    let smtp = null;
    let integrations = null;
    try {
        smtp = await locals.api.admin.getEmailSettings();
    } catch (err) {
        console.error('Failed to load SMTP settings:', err);
    }
    try {
        integrations = await locals.api.admin.getIntegrationSettings();
    } catch (err) {
        console.error('Failed to load integration settings:', err);
    }
    return { smtp, integrations };
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
    },

    saveTransport: async ({ locals, request }) => {
        const fd = await request.formData();
        const mail_transport = String(fd.get('mail_transport') ?? '').trim() as 'smtp' | 'resend';
        if (mail_transport !== 'smtp' && mail_transport !== 'resend') {
            return fail(400, { saveTransport: { error: m.error_select_transport() } });
        }
        try {
            await locals.api.admin.updateMailTransport(mail_transport);
            return { saveTransport: { ok: true, transport: mail_transport } };
        } catch (err) {
            console.error('Mail transport update failed:', err);
            return fail(500, { saveTransport: { error: m.error_transport_update() } });
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

    saveMode: async ({ locals, request }) => {
        const fd = await request.formData();
        const mail_mode = String(fd.get('mail_mode') ?? '').trim() as 'smtp' | 'manual_links';
        if (mail_mode !== 'smtp' && mail_mode !== 'manual_links') {
            return fail(400, { saveMode: { error: m.error_mail_mode() } });
        }
        try {
            await locals.api.admin.updateMailMode(mail_mode);
            return { saveMode: { ok: true, mode: mail_mode } };
        } catch (err) {
            console.error('Mail mode update failed:', err);
            return fail(500, { saveMode: { error: m.error_mail_mode_update() } });
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
    },

    saveSlack: async ({ locals, request }) => {
        const fd = await request.formData();
        const enabled = fd.get('enabled') === 'on';
        const webhook_url = String(fd.get('webhook_url') ?? '').trim();
        const events = parseEvents(fd);
        try {
            await locals.api.admin.updateSlackIntegration({ enabled, webhook_url, events });
            return { saveSlack: { ok: true } };
        } catch (err: unknown) {
            const code = err instanceof Error ? err.message : '';
            if (code === 'SMTP_ENCRYPTION_KEY_MISSING') {
                return fail(400, {
                    saveSlack: {
                        error: m.error_slack_key()
                    }
                });
            }
            console.error('Slack save failed:', err);
            return fail(500, {
                saveSlack: { error: code.startsWith('slack') || code.includes('URL') ? code : m.error_slack_save() }
            });
        }
    },

    saveGotify: async ({ locals, request }) => {
        const fd = await request.formData();
        const enabled = fd.get('enabled') === 'on';
        const url = String(fd.get('url') ?? '').trim();
        const token = String(fd.get('token') ?? '');
        const events = parseEvents(fd);
        try {
            await locals.api.admin.updateGotifyIntegration({ enabled, url, token, events });
            return { saveGotify: { ok: true } };
        } catch (err: unknown) {
            const code = err instanceof Error ? err.message : '';
            if (code === 'SMTP_ENCRYPTION_KEY_MISSING') {
                return fail(400, {
                    saveGotify: {
                        error: m.error_gotify_key()
                    }
                });
            }
            console.error('Gotify save failed:', err);
            return fail(500, {
                saveGotify: { error: m.error_gotify_save() }
            });
        }
    },

    saveWebhook: async ({ locals, request }) => {
        const fd = await request.formData();
        const enabled = fd.get('enabled') === 'on';
        const url = String(fd.get('url') ?? '').trim();
        const bearer_token = String(fd.get('bearer_token') ?? '');
        const events = parseEvents(fd);
        try {
            await locals.api.admin.updateWebhookIntegration({ enabled, url, bearer_token, events });
            return { saveWebhook: { ok: true } };
        } catch (err: unknown) {
            const code = err instanceof Error ? err.message : '';
            if (code === 'SMTP_ENCRYPTION_KEY_MISSING') {
                return fail(400, {
                    saveWebhook: {
                        error: m.error_webhook_key()
                    }
                });
            }
            console.error('Webhook save failed:', err);
            return fail(500, {
                saveWebhook: { error: m.error_webhook_save() }
            });
        }
    },

    testSlack: async ({ locals }) => {
        try {
            await locals.api.admin.sendIntegrationTest('slack');
            return { testSlack: { ok: true } };
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : '';
            if (msg === 'INTEGRATION_TEST_NOT_READY') {
                return fail(400, { testSlack: { error: m.error_slack_test_first() } });
            }
            console.error('Slack test failed:', err);
            return fail(500, { testSlack: { error: m.error_slack_test() } });
        }
    },

    testGotify: async ({ locals }) => {
        try {
            await locals.api.admin.sendIntegrationTest('gotify');
            return { testGotify: { ok: true } };
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : '';
            if (msg === 'INTEGRATION_TEST_NOT_READY') {
                return fail(400, { testGotify: { error: m.error_gotify_test_first() } });
            }
            console.error('Gotify test failed:', err);
            return fail(500, { testGotify: { error: m.error_gotify_test() } });
        }
    },

    testWebhook: async ({ locals }) => {
        try {
            await locals.api.admin.sendIntegrationTest('webhook');
            return { testWebhook: { ok: true } };
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : '';
            if (msg === 'INTEGRATION_TEST_NOT_READY') {
                return fail(400, { testWebhook: { error: m.error_webhook_test_first() } });
            }
            console.error('Webhook test failed:', err);
            return fail(500, { testWebhook: { error: m.error_webhook_test() } });
        }
    }
};
