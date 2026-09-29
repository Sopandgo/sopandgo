import type { Client } from './client';
import type { 
    User, 
    UserRole, 
    AdminIntegrity, 
    AuditLogResponse,
    AuditFilterOptionsResponse,
    PublicSmtpSettings,
    PublicIntegrationSettings,
    IntegrationEvent,
    AdminCreateUserResponse,
    AdminPasswordResetResponse,
    BackupStatus,
    BackupImportValidation,
    BackupApplyResponse
} from './types';

export const admin = (client: Client) => ({
    
    // --- User Management ---

    /**
     * List all registered users.
     */
    listUsers: async (): Promise<User[]> => {
        const res = await client.fetch('/admin/users', {
            method: 'GET'
        });
        if (!res.ok) throw new Error('ADMIN_FETCH_USERS_FAILED');
        return await res.json();
    },

    /**
     * Create a new user account.
     */
    createUser: async (payload: { 
        display_name: string; 
        email: string; 
        role: UserRole;
    }): Promise<AdminCreateUserResponse> => {
        const res = await client.fetch('/admin/users/register', {
            method: 'POST',
            body: payload
        });
        if (!res.ok) throw new Error('USER_CREATION_FAILED');
        return await res.json();
    },

    /**
     * Triggers a password reset for a user account.
     */
    triggerPasswordReset: async (userId: string): Promise<AdminPasswordResetResponse> => {
        const res = await client.fetch(`/admin/users/${userId}/reset-password`, {
            method: 'POST',
        });
        if (!res.ok) throw new Error('TRIGGER_RESET_FAILED');
        return await res.json();
    },

    /**
     * Update a user's role.
     */
    updateRole: async (userId: string, newRole: UserRole): Promise<void> => {
        const res = await client.fetch(`/admin/users/${userId}/update-role`, {
            method: 'PATCH',
            body: { new_role: newRole }
        });
        if (!res.ok) throw new Error('UPDATE_ROLE_FAILED');
    },

    /**
     * Enable or Disable a user.
     */
    updateStatus: async (userId: string, active: boolean): Promise<void> => {
        const res = await client.fetch(`/admin/users/${userId}/update-status`, {
            method: 'PATCH',
            body: { active }
        });
        if (!res.ok) {
            if (res.status === 403) throw new Error('CANNOT_DISABLE_SELF');
            throw new Error('UPDATE_STATUS_FAILED');
        }
    },

    // --- Session Management ---

    /**
     * List all active system sessions.
     */
    listSessions: async (): Promise<any[]> => {
        const res = await client.fetch('/admin/sessions', {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_SESSIONS_FAILED');
        return await res.json();
    },

    /**
     * Revoke all sessions for a specific user.
     */
    revokeUserSessions: async (userId: string): Promise<void> => {
        const res = await client.fetch(`/admin/users/${userId}/session`, {
            method: 'DELETE'
        });
        if (!res.ok) throw new Error('REVOKE_FAILED');
    },

    /**
     * System-wide "Panic Button": Revoke ALL sessions for everyone.
     */
    revokeAllSessions: async (reason: string): Promise<void> => {
        if (!reason || reason.trim() === '') {
            throw new Error('REASON_REQUIRED');
        }

        const res = await client.fetch('/admin/sessions', {
            method: 'DELETE',
            body: { reason }
        });
        if (!res.ok) throw new Error('GLOBAL_REVOKE_FAILED');
    },

    // --- System Health ---

    /**
     * Fetch system audit logs.
     */
    getAuditLogs: async (
        limit = 50,
        offset = 0,
        filters?: {
            type?: string;
            entity_type?: string;
            actor_user_id?: string;
            sop_id?: string;
        }
    ): Promise<AuditLogResponse> => {
        const params = new URLSearchParams({
            limit: limit.toString(),
            offset: offset.toString()
        });

        if (filters?.type) {
            params.append('type', filters.type);
        }
        if (filters?.entity_type) {
            params.append('entity_type', filters.entity_type);
        }
        if (filters?.actor_user_id) {
            params.append('actor_user_id', filters.actor_user_id);
        }
        if (filters?.sop_id) {
            params.append('sop_id', filters.sop_id);
        }

        const res = await client.fetch(`/admin/audit-logs?${params.toString()}`, { 
            method: 'GET' 
        });

        if (!res.ok) throw new Error('FETCH_LOGS_FAILED');

        // Automatically matches the AuditLogResponse interface
        return await res.json(); 
    },

    getAuditFilterOptions: async (): Promise<AuditFilterOptionsResponse> => {
        const res = await client.fetch('/admin/audit-logs/options', {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_AUDIT_FILTER_OPTIONS_FAILED');
        return await res.json();
    },

    /**
     * Global system-wide integrity check.
     */
    checkIntegrity: async (): Promise<AdminIntegrity> => {
        const res = await client.fetch('/admin/integrity', { 
            method: 'GET'
        });
        if (!res.ok) throw new Error('INTEGRITY_CHECK_FAILED');
        return await res.json();
    },

    // --- Email / SMTP (admin UI) ---

    /** Loads outbound email settings (SMTP + Resend + mail mode). Prefer over legacy path. */
    getEmailSettings: async (): Promise<PublicSmtpSettings> => {
        const res = await client.fetch('/admin/settings/email', { method: 'GET' });
        if (!res.ok) throw new Error('EMAIL_SETTINGS_FETCH_FAILED');
        return await res.json();
    },

    updateSmtpSettings: async (payload: {
        host: string;
        port: string;
        username: string;
        from_address: string;
        password: string;
    }): Promise<void> => {
        const res = await client.fetch('/admin/settings/smtp', {
            method: 'PUT',
            body: payload
        });
        if (res.status === 412) throw new Error('SMTP_ENCRYPTION_KEY_MISSING');
        if (res.status === 400) {
            const t = await res.text();
            throw new Error(t || 'SMTP_SETTINGS_VALIDATION_FAILED');
        }
        if (!res.ok) throw new Error('SMTP_SETTINGS_SAVE_FAILED');
    },

    sendSmtpTest: async (to: string): Promise<void> => {
        const res = await client.fetch('/admin/settings/smtp/test', {
            method: 'POST',
            body: { to }
        });
        if (res.status === 412) throw new Error('MAIL_TEST_NOT_READY');
        if (res.status === 502) {
            const t = await res.text();
            throw new Error(t || 'SMTP_TEST_SEND_FAILED');
        }
        if (!res.ok) throw new Error('SMTP_TEST_FAILED');
    },

    updateMailTransport: async (mail_transport: 'smtp' | 'resend'): Promise<void> => {
        const res = await client.fetch('/admin/settings/mail-transport', {
            method: 'PATCH',
            body: { mail_transport }
        });
        if (res.status === 400) {
            const t = await res.text();
            throw new Error(t || 'MAIL_TRANSPORT_INVALID');
        }
        if (!res.ok) throw new Error('MAIL_TRANSPORT_UPDATE_FAILED');
    },

    updateResendSettings: async (payload: { from_address: string; api_key: string }): Promise<void> => {
        const res = await client.fetch('/admin/settings/resend', {
            method: 'PUT',
            body: payload
        });
        if (res.status === 412) throw new Error('SMTP_ENCRYPTION_KEY_MISSING');
        if (res.status === 400) {
            const t = await res.text();
            throw new Error(t || 'RESEND_SETTINGS_VALIDATION_FAILED');
        }
        if (!res.ok) throw new Error('RESEND_SETTINGS_SAVE_FAILED');
    },

    updateMailMode: async (mail_mode: 'smtp' | 'manual_links'): Promise<void> => {
        const res = await client.fetch('/admin/settings/mail-mode', {
            method: 'PATCH',
            body: { mail_mode }
        });
        if (res.status === 400) {
            const t = await res.text();
            throw new Error(t || 'MAIL_MODE_INVALID');
        }
        if (!res.ok) throw new Error('MAIL_MODE_UPDATE_FAILED');
    },

    // --- Integrations (Slack / Gotify / webhook) ---

    getIntegrationSettings: async (): Promise<PublicIntegrationSettings> => {
        const res = await client.fetch('/admin/settings/integrations', { method: 'GET' });
        if (!res.ok) throw new Error('INTEGRATION_SETTINGS_FETCH_FAILED');
        return await res.json();
    },

    updateSlackIntegration: async (payload: {
        enabled: boolean;
        webhook_url: string;
        events: IntegrationEvent[];
    }): Promise<void> => {
        const res = await client.fetch('/admin/settings/integrations/slack', {
            method: 'PUT',
            body: payload
        });
        if (res.status === 412) throw new Error('SMTP_ENCRYPTION_KEY_MISSING');
        if (res.status === 400) {
            const t = await res.text();
            throw new Error(t || 'SLACK_SETTINGS_VALIDATION_FAILED');
        }
        if (!res.ok) throw new Error('SLACK_SETTINGS_SAVE_FAILED');
    },

    updateGotifyIntegration: async (payload: {
        enabled: boolean;
        url: string;
        token: string;
        events: IntegrationEvent[];
    }): Promise<void> => {
        const res = await client.fetch('/admin/settings/integrations/gotify', {
            method: 'PUT',
            body: payload
        });
        if (res.status === 412) throw new Error('SMTP_ENCRYPTION_KEY_MISSING');
        if (res.status === 400) {
            const t = await res.text();
            throw new Error(t || 'GOTIFY_SETTINGS_VALIDATION_FAILED');
        }
        if (!res.ok) throw new Error('GOTIFY_SETTINGS_SAVE_FAILED');
    },

    updateWebhookIntegration: async (payload: {
        enabled: boolean;
        url: string;
        bearer_token: string;
        events: IntegrationEvent[];
    }): Promise<void> => {
        const res = await client.fetch('/admin/settings/integrations/webhook', {
            method: 'PUT',
            body: payload
        });
        if (res.status === 412) throw new Error('SMTP_ENCRYPTION_KEY_MISSING');
        if (res.status === 400) {
            const t = await res.text();
            throw new Error(t || 'WEBHOOK_SETTINGS_VALIDATION_FAILED');
        }
        if (!res.ok) throw new Error('WEBHOOK_SETTINGS_SAVE_FAILED');
    },

    sendIntegrationTest: async (channel: 'slack' | 'gotify' | 'webhook'): Promise<void> => {
        const res = await client.fetch(`/admin/settings/integrations/${channel}/test`, {
            method: 'POST',
            body: {}
        });
        if (res.status === 412) throw new Error('INTEGRATION_TEST_NOT_READY');
        if (res.status === 502) {
            const t = await res.text();
            throw new Error(t || 'INTEGRATION_TEST_SEND_FAILED');
        }
        if (!res.ok) throw new Error('INTEGRATION_TEST_FAILED');
    },

    getBackupStatus: async (): Promise<BackupStatus> => {
        const res = await client.fetch('/admin/backups/status', { method: 'GET' });
        if (!res.ok) throw new Error('BACKUP_STATUS_FAILED');
        return await res.json();
    },

    validateBackupImport: async (backupFile: File): Promise<BackupImportValidation> => {
        const fd = new FormData();
        fd.set('backup', backupFile);

        const res = await client.fetch('/admin/backups/import', {
            method: 'POST',
            body: fd
        });
        if (res.status === 422) {
            const t = await res.text();
            throw new Error(t || 'BACKUP_IMPORT_INCOMPATIBLE');
        }
        if (res.status === 409) {
            const t = await res.text();
            if (t.includes('waiting for restart')) throw new Error('BACKUP_PENDING_RESTART');
            throw new Error('BACKUP_BUSY');
        }
        if (res.status === 400) throw new Error('BACKUP_INVALID_ARCHIVE');
        if (!res.ok) throw new Error('BACKUP_IMPORT_VALIDATION_FAILED');
        return await res.json();
    },

    applyBackup: async (backupFile: File, confirmation: string): Promise<BackupApplyResponse> => {
        const fd = new FormData();
        fd.set('backup', backupFile);
        fd.set('confirmation', confirmation);

        const res = await client.fetch('/admin/backups/apply', {
            method: 'POST',
            body: fd
        });
        if (res.status === 422) {
            const t = await res.text();
            throw new Error(t || 'BACKUP_APPLY_INCOMPATIBLE');
        }
        if (res.status === 409) {
            const t = await res.text();
            if (t.includes('waiting for restart')) throw new Error('BACKUP_PENDING_RESTART');
            throw new Error('BACKUP_BUSY');
        }
        if (res.status === 400) {
            const t = await res.text();
            if (t.includes('confirmation must equal')) throw new Error('BACKUP_CONFIRMATION_REQUIRED');
            throw new Error(t || 'BACKUP_APPLY_INVALID');
        }
        if (!res.ok) throw new Error('BACKUP_APPLY_FAILED');
        return await res.json();
    }
});