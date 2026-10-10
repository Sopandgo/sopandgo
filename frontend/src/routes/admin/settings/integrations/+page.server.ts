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
    let integrations = null;
    try {
        integrations = await locals.api.admin.getIntegrationSettings();
    } catch (err) {
        console.error('Failed to load integration settings:', err);
    }
    return { integrations };
};

export const actions: Actions = {
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

    // The header toggle: flips only `enabled`. Plain URLs and events are sent back
    // as stored (the backend overwrites them); blank secrets keep the stored ones.
    setEnabled: async ({ locals, request }) => {
        const fd = await request.formData();
        const channel = String(fd.get('channel') ?? '');
        const enabled = fd.get('enabled') === 'on';
        if (channel !== 'slack' && channel !== 'gotify' && channel !== 'webhook') {
            return fail(400, { setEnabled: { channel, error: m.settings_integration_toggle_error() } });
        }
        try {
            const current = await locals.api.admin.getIntegrationSettings();
            const { events, url = '' } = current[channel];
            if (channel === 'slack') {
                await locals.api.admin.updateSlackIntegration({ enabled, webhook_url: '', events });
            } else if (channel === 'gotify') {
                await locals.api.admin.updateGotifyIntegration({ enabled, url, token: '', events });
            } else {
                await locals.api.admin.updateWebhookIntegration({ enabled, url, bearer_token: '', events });
            }
            return { setEnabled: { channel, ok: true } };
        } catch (err: unknown) {
            console.error(`${channel} toggle failed:`, err);
            return fail(500, { setEnabled: { channel, error: m.settings_integration_toggle_error() } });
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
