import { fail } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';
import type { BackupS3Settings, BackupS3SettingsInput } from '#lib/sdk/types.js';

export const load: PageServerLoad = async ({ locals }) => {
    const [backupStatus, s3Settings] = await Promise.all([
        locals.api.admin.getBackupStatus().catch((err) => {
            console.error('Failed to load backup status:', err);
            return { locked: false, message: '', pending_restore: false };
        }),
        locals.api.admin.getBackupS3Settings().catch((err): BackupS3Settings | null => {
            console.error('Failed to load S3 backup settings:', err);
            return null;
        })
    ]);
    return { backupStatus, s3Settings };
};

function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : '';
}

function nonNegativeInt(v: FormDataEntryValue | null): number {
    const n = Number.parseInt(String(v ?? '').trim(), 10);
    return Number.isFinite(n) ? n : 0;
}

// Backend validation and S3 errors arrive as readable text; codes map to messages.
function s3SaveError(code: string): string {
    if (code === 'SECRET_ENCRYPTION_KEY_MISSING') return m.error_backup_s3_key();
    if (code.startsWith('invalid S3 backup settings') || code.startsWith('settings saved')) return code;
    return m.error_backup_s3_save();
}

export const actions: Actions = {
    saveS3: async ({ locals, request }) => {
        const fd = await request.formData();
        const input: BackupS3SettingsInput = {
            enabled: fd.get('enabled') === 'on',
            bucket: String(fd.get('bucket') ?? '').trim(),
            region: String(fd.get('region') ?? '').trim(),
            key_prefix: String(fd.get('key_prefix') ?? '').trim(),
            endpoint: String(fd.get('endpoint') ?? '').trim(),
            use_path_style: fd.get('use_path_style') === 'on',
            access_key_id: String(fd.get('access_key_id') ?? '').trim(),
            secret_access_key: String(fd.get('secret_access_key') ?? ''),
            interval: String(fd.get('interval') ?? '').trim(),
            retention_max: nonNegativeInt(fd.get('retention_max')),
            retention_days: nonNegativeInt(fd.get('retention_days'))
        };
        try {
            await locals.api.admin.updateBackupS3Settings(input);
            return { saveS3: { ok: true } };
        } catch (err: unknown) {
            console.error('S3 backup settings save failed:', err);
            return fail(400, { saveS3: { error: s3SaveError(errorMessage(err)) } });
        }
    },

    // The header toggle: flips only `enabled` and sends the rest back as saved
    // (a blank secret keeps the stored one).
    setS3Enabled: async ({ locals, request }) => {
        const fd = await request.formData();
        const enabled = fd.get('enabled') === 'on';
        try {
            const s = await locals.api.admin.getBackupS3Settings();
            await locals.api.admin.updateBackupS3Settings({
                enabled,
                bucket: s.bucket,
                region: s.region,
                key_prefix: s.key_prefix,
                endpoint: s.endpoint,
                use_path_style: s.use_path_style,
                access_key_id: s.access_key_id,
                secret_access_key: '',
                interval: s.interval,
                retention_max: s.retention_max,
                retention_days: s.retention_days
            });
            return { setS3Enabled: { ok: true } };
        } catch (err: unknown) {
            console.error('S3 backup toggle failed:', err);
            return fail(400, { setS3Enabled: { error: m.error_backup_s3_toggle() } });
        }
    },

    testS3: async ({ locals }) => {
        try {
            await locals.api.admin.testBackupS3();
            return { testS3: { ok: true } };
        } catch (err: unknown) {
            const code = errorMessage(err);
            const error =
                code === 'BACKUP_S3_NOT_CONFIGURED'
                    ? m.error_backup_s3_not_configured()
                    : code.startsWith('S3 connection test failed')
                      ? code
                      : m.error_backup_s3_test();
            return fail(400, { testS3: { error } });
        }
    },

    runS3: async ({ locals }) => {
        try {
            const { object_key } = await locals.api.admin.runBackupS3Now();
            return { runS3: { ok: true, objectKey: object_key } };
        } catch (err: unknown) {
            const code = errorMessage(err);
            let error: string = m.error_backup_s3_run();
            if (code === 'BACKUP_S3_OFF') error = m.error_backup_s3_off();
            if (code === 'BACKUP_BUSY') error = m.error_backup_busy();
            if (code.startsWith('S3 backup failed')) error = code;
            return fail(400, { runS3: { error } });
        }
    },

    importBackup: async ({ locals, request }) => {
        const fd = await request.formData();
        const file = fd.get('backup_file');
        if (!(file instanceof File) || file.size <= 0) {
            return fail(400, {
                importBackup: { error: m.error_backup_choose() }
            });
        }
        try {
            const result = await locals.api.admin.validateBackupImport(file);
            return { importBackup: { ok: true, manifest: result.manifest, note: result.note } };
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : 'BACKUP_IMPORT_VALIDATION_FAILED';
            let pretty: string = m.error_backup_validate();
            if (msg === 'BACKUP_BUSY') pretty = m.error_backup_busy();
            if (msg === 'BACKUP_PENDING_RESTART') pretty = m.error_backup_pending();
            if (msg === 'BACKUP_INVALID_ARCHIVE') pretty = m.error_backup_invalid();
            if (msg.includes('unsupported backup format')) pretty = msg;
            if (msg.includes('backup schema is newer')) pretty = msg;
            return fail(422, {
                importBackup: { error: pretty }
            });
        }
    },

    applyBackup: async ({ locals, request }) => {
        const fd = await request.formData();
        const file = fd.get('backup_file');
        const confirmation = String(fd.get('confirmation') ?? '').trim();

        if (!(file instanceof File) || file.size <= 0) {
            return fail(400, {
                applyBackup: { error: m.error_backup_choose() }
            });
        }
        try {
            const result = await locals.api.admin.applyBackup(file, confirmation);
            return {
                applyBackup: {
                    ok: true,
                    note: result.note,
                    requires_restart: result.requires_restart,
                    pre_apply_backup: result.pre_apply_backup,
                    manifest: result.manifest
                }
            };
        } catch (err: unknown) {
            const msg = err instanceof Error ? err.message : 'BACKUP_APPLY_FAILED';
            let pretty: string = m.error_backup_stage();
            if (msg === 'BACKUP_BUSY') pretty = m.error_backup_busy();
            if (msg === 'BACKUP_PENDING_RESTART') pretty = m.error_backup_pending();
            if (msg === 'BACKUP_CONFIRMATION_REQUIRED') pretty = m.error_backup_confirm();
            if (msg.includes('unsupported backup format')) pretty = msg;
            if (msg.includes('backup schema is newer')) pretty = msg;
            return fail(422, {
                applyBackup: { error: pretty }
            });
        }
    }
};
