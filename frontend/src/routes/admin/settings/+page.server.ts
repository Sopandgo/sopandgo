import { fail } from '@sveltejs/kit';
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
    saveTransport: async ({ locals, request }) => {
        const fd = await request.formData();
        const mail_transport = String(fd.get('mail_transport') ?? '').trim() as 'smtp' | 'resend';
        if (mail_transport !== 'smtp' && mail_transport !== 'resend') {
            return fail(400, { saveTransport: { error: 'Select SMTP or Resend.' } });
        }
        try {
            await locals.api.admin.updateMailTransport(mail_transport);
            return { saveTransport: { ok: true, transport: mail_transport } };
        } catch (err) {
            console.error('Mail transport update failed:', err);
            return fail(500, { saveTransport: { error: 'Could not update outbound transport.' } });
        }
    },

    saveResend: async ({ locals, request }) => {
        const fd = await request.formData();
        const from_address = String(fd.get('resend_from_address') ?? '').trim();
        const api_key = String(fd.get('resend_api_key') ?? '');

        if (!from_address) {
            return fail(400, {
                saveResend: { error: 'From address is required.' },
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
                        error:
                            'The server is not configured with SMTP_SECRET_ENCRYPTION_KEY. The same key encrypts stored API secrets; add a 32-byte key to the environment and restart.'
                    },
                    resendValues: { from_address }
                });
            }
            console.error('Resend save failed:', err);
            return fail(500, {
                saveResend: { error: 'Could not save Resend settings.' },
                resendValues: { from_address }
            });
        }
    },

    saveMode: async ({ locals, request }) => {
        const fd = await request.formData();
        const mail_mode = String(fd.get('mail_mode') ?? '').trim() as 'smtp' | 'manual_links';
        if (mail_mode !== 'smtp' && mail_mode !== 'manual_links') {
            return fail(400, { saveMode: { error: 'Select a valid mail mode.' } });
        }
        try {
            await locals.api.admin.updateMailMode(mail_mode);
            return { saveMode: { ok: true, mode: mail_mode } };
        } catch (err) {
            console.error('Mail mode update failed:', err);
            return fail(500, { saveMode: { error: 'Could not update mail mode.' } });
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
                save: { error: 'Host, port, username, and from address are required.' },
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
                        error:
                            'The server is not configured with SMTP_SECRET_ENCRYPTION_KEY. Add a 32-byte key to the environment (e.g. openssl rand -base64 32) and restart.'
                    },
                    values: { host, port, username, from_address }
                });
            }
            console.error('SMTP save failed:', err);
            return fail(500, {
                save: { error: 'Could not save SMTP settings.' },
                values: { host, port, username, from_address }
            });
        }
    },

    test: async ({ locals, request }) => {
        const fd = await request.formData();
        const test_to = String(fd.get('test_to') ?? '').trim().toLowerCase();

        if (!test_to || !test_to.includes('@')) {
            return fail(400, { test: { error: 'Enter a valid recipient email address.' } });
        }

        try {
            await locals.api.admin.sendSmtpTest(test_to);
            return { test: { ok: true } };
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : 'SMTP_TEST_FAILED';
            if (msg === 'MAIL_TEST_NOT_READY' || msg === 'SMTP_NOT_READY') {
                return fail(400, {
                    test: {
                        error:
                            'Complete email settings for the selected transport (SMTP or Resend) and ensure SMTP_SECRET_ENCRYPTION_KEY is set on the server.'
                    }
                });
            }
            console.error('SMTP test failed:', err);
            return fail(500, {
                test: { error: msg === 'SMTP_TEST_FAILED' ? 'Could not send test email.' : msg }
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
                        error:
                            'SMTP_SECRET_ENCRYPTION_KEY is required to store the Slack webhook URL. Set it and restart.'
                    }
                });
            }
            console.error('Slack save failed:', err);
            return fail(500, {
                saveSlack: { error: code.startsWith('slack') || code.includes('URL') ? code : 'Could not save Slack settings.' }
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
                        error:
                            'SMTP_SECRET_ENCRYPTION_KEY is required to store the Gotify token. Set it and restart.'
                    }
                });
            }
            console.error('Gotify save failed:', err);
            return fail(500, {
                saveGotify: { error: 'Could not save Gotify settings.' }
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
                        error:
                            'SMTP_SECRET_ENCRYPTION_KEY is required to store a webhook bearer token. Set it and restart.'
                    }
                });
            }
            console.error('Webhook save failed:', err);
            return fail(500, {
                saveWebhook: { error: 'Could not save webhook settings.' }
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
                return fail(400, { testSlack: { error: 'Save a Slack webhook URL first.' } });
            }
            console.error('Slack test failed:', err);
            return fail(500, { testSlack: { error: 'Could not send Slack test notification.' } });
        }
    },

    testGotify: async ({ locals }) => {
        try {
            await locals.api.admin.sendIntegrationTest('gotify');
            return { testGotify: { ok: true } };
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : '';
            if (msg === 'INTEGRATION_TEST_NOT_READY') {
                return fail(400, { testGotify: { error: 'Save Gotify URL and token first.' } });
            }
            console.error('Gotify test failed:', err);
            return fail(500, { testGotify: { error: 'Could not send Gotify test notification.' } });
        }
    },

    testWebhook: async ({ locals }) => {
        try {
            await locals.api.admin.sendIntegrationTest('webhook');
            return { testWebhook: { ok: true } };
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : '';
            if (msg === 'INTEGRATION_TEST_NOT_READY') {
                return fail(400, { testWebhook: { error: 'Save a webhook URL first.' } });
            }
            console.error('Webhook test failed:', err);
            return fail(500, { testWebhook: { error: 'Could not send webhook test notification.' } });
        }
    }
};
