import { fail } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { Actions } from './$types';

// The check runs on demand only: each run writes an audit event and may notify integrations.
export const actions: Actions = {
    run: async ({ locals }) => {
        try {
            const report = await locals.api.admin.checkIntegrity();
            return { report };
        } catch (err) {
            console.error('System integrity check failed:', err);
            return fail(500, { error: m.integrity_admin_run_failed() });
        }
    }
};
