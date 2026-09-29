import { fail, error } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
    try {
        // Use SDK to list system-wide sessions
        const sessions = await locals.api.admin.listSessions();
        return { sessions };
    } catch (err) {
        console.error('Failed to load sessions:', err);
        throw error(500, 'Failed to load system sessions');
    }
};

export const actions: Actions = {
    revokeUserSessions: async ({ locals, request }) => {
        const fd = await request.formData();
        const user_id = String(fd.get('user_id') ?? '').trim();

        if (!user_id) {
            return fail(400, { revokeUserSessions: { error: 'Missing user_id.' } });
        }

        try {
            // Use SDK to revoke sessions for a specific user
            await locals.api.admin.revokeUserSessions(user_id);
            return { revokeUserSessions: { ok: true, user_id } };
        } catch (err) {
            console.error('Revoke user sessions failed:', err);
            return fail(500, {
                revokeUserSessions: { error: 'Failed to revoke user sessions.' }
            });
        }
    },

    revokeAllSessions: async ({ locals, request }) => {
        const fd = await request.formData();
        const reason = String(fd.get('reason') ?? '').trim();

        if (!reason) {
            return fail(400, { revokeAllSessions: { error: 'Reason is required for global revocation.' } });
        }

        try {
            // Use SDK "Panic Button" to revoke ALL sessions
            await locals.api.admin.revokeAllSessions(reason);
            return { revokeAllSessions: { ok: true } };
        } catch (err: any) {
            console.error('Global revocation failed:', err);
            
            // Pass through specific SDK errors if they exist
            const message = err.message === 'REASON_REQUIRED' 
                ? 'A valid reason must be provided.' 
                : 'Failed to perform global session revocation.';

            return fail(500, { revokeAllSessions: { error: message } });
        }
    }
};