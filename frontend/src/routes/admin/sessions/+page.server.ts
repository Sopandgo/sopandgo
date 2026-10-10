import { fail, error } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
    try {
        // Use SDK to list system-wide sessions
        const sessions = await locals.api.admin.listSessions();
        return { sessions };
    } catch (err) {
        console.error('Failed to load sessions:', err);
        throw error(500, m.error_sessions_load());
    }
};

export const actions: Actions = {
    revokeUserSessions: async ({ locals, request }) => {
        const fd = await request.formData();
        const user_id = String(fd.get('user_id') ?? '').trim();

        if (!user_id) {
            return fail(400, { revokeUserSessions: { error: m.error_missing_user() } });
        }

        try {
            // Use SDK to revoke sessions for a specific user
            await locals.api.admin.revokeUserSessions(user_id);
            return { revokeUserSessions: { ok: true, user_id } };
        } catch (err) {
            console.error('Revoke user sessions failed:', err);
            return fail(500, {
                revokeUserSessions: { error: m.error_revoke_user_failed() }
            });
        }
    },

    revokeAllSessions: async ({ locals, request }) => {
        const fd = await request.formData();
        const reason = String(fd.get('reason') ?? '').trim();

        if (!reason) {
            return fail(400, { revokeAllSessions: { error: m.error_revoke_reason() } });
        }

        try {
            // Use SDK "Panic Button" to revoke ALL sessions
            await locals.api.admin.revokeAllSessions(reason);
            return { revokeAllSessions: { ok: true } };
        } catch (err: any) {
            console.error('Global revocation failed:', err);
            
            // Pass through specific SDK errors if they exist
            const message = err.message === 'REASON_REQUIRED' 
                ? m.error_revoke_reason_valid() 
                : m.error_revoke_all_failed();

            return fail(500, { revokeAllSessions: { error: message } });
        }
    }
};
