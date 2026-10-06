import { fail } from '@sveltejs/kit';
import * as m from '$lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
    try {
        const backupStatus = await locals.api.admin.getBackupStatus();
        return { backupStatus };
    } catch (err) {
        console.error('Failed to load backup status:', err);
        return { backupStatus: { locked: false, message: '', pending_restore: false } };
    }
};

export const actions: Actions = {
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
