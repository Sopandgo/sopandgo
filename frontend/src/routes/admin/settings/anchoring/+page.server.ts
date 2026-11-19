import { fail } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';
import type {
    AuditAnchoringAuth,
    AuditAnchoringInput,
    AuditAnchoringStatus,
    BackupS3Settings
} from '#lib/sdk/types.js';

export const load: PageServerLoad = async ({ locals }) => {
    const [status, s3Settings] = await Promise.all([
        locals.api.admin.getAuditAnchoring().catch((err): AuditAnchoringStatus | null => {
            console.error('Failed to load audit anchoring status:', err);
            return null;
        }),
        locals.api.admin.getBackupS3Settings().catch((err): BackupS3Settings | null => {
            console.error('Failed to load S3 backup settings:', err);
            return null;
        })
    ]);
    return { status, s3Configured: s3Settings?.configured ?? false };
};

function errorMessage(err: unknown, fallback: string): string {
    return err instanceof Error && err.message && !/^[A-Z_]+$/.test(err.message) ? err.message : fallback;
}

function authType(v: FormDataEntryValue | null): AuditAnchoringAuth {
    const s = String(v ?? 'none');
    return s === 'basic' || s === 'bearer' ? s : 'none';
}

export const actions: Actions = {
    // One form for switching on and for changes. The first submit of a new TSA
    // comes back with what to confirm; the confirm button resubmits the same
    // fields with the fingerprint and the accepted rows.
    save: async ({ locals, request }) => {
        const fd = await request.formData();
        const input: AuditAnchoringInput = {
            enabled: true,
            tsa_url: String(fd.get('tsa_url') ?? '').trim(),
            auth_type: authType(fd.get('auth_type')),
            username: String(fd.get('username') ?? '').trim(),
            secret: String(fd.get('secret') ?? ''),
            interval: String(fd.get('interval') ?? '').trim(),
            pinned_ca_pem: String(fd.get('pinned_ca_pem') ?? '').trim(),
            repin: fd.get('repin') === 'on',
            confirm_fingerprint: String(fd.get('confirm_fingerprint') ?? ''),
            accept_broken: fd.getAll('accept_broken').map((v) => Number(v)).filter(Number.isFinite)
        };
        try {
            const result = await locals.api.admin.saveAuditAnchoring(input);
            if (!result.ok) {
                return { save: { confirm: result.confirm } };
            }
            return { save: { ok: true, wasEnabled: fd.get('was_enabled') === 'on' } };
        } catch (err: unknown) {
            console.error('Audit anchoring save failed:', err);
            return fail(400, { save: { error: errorMessage(err, m.error_anchoring_save()) } });
        }
    },

    // Switching off keeps the saved TSA (a blank secret keeps the stored one);
    // the server takes a final anchor first.
    disable: async ({ locals }) => {
        try {
            const s = await locals.api.admin.getAuditAnchoring();
            await locals.api.admin.saveAuditAnchoring({
                enabled: false,
                tsa_url: s.tsa_url,
                auth_type: s.auth_type,
                username: s.username,
                secret: '',
                interval: s.interval
            });
            return { disable: { ok: true } };
        } catch (err: unknown) {
            console.error('Audit anchoring switch-off failed:', err);
            return fail(400, { disable: { error: errorMessage(err, m.error_anchoring_save()) } });
        }
    },

    run: async ({ locals }) => {
        try {
            const res = await locals.api.admin.runAuditAnchoring();
            return { run: { ok: true, skipped: res.skipped, row: res.anchor?.event_id, warning: res.warning } };
        } catch (err: unknown) {
            return fail(400, { run: { error: errorMessage(err, m.error_anchoring_run()) } });
        }
    }
};
