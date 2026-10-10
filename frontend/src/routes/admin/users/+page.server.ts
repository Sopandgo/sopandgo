import { fail } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';
import type { UserRole } from '#lib/sdk/types.js';

// Valid roles set for validation
const ROLES = new Set(['admin', 'editor', 'approver', 'auditor', 'viewer']);

/** @param {string} email */
function looksLikeEmail(email: string) {
    return email.includes('@') && email.includes('.');
}

export const load: PageServerLoad = async ({ locals }) => {
    try {
        // Use SDK to list users
        const users = await locals.api.admin.listUsers();
        const smtp = await locals.api.admin.getEmailSettings();
        return { users, smtp };
    } catch (err) {
        console.error('Failed to load users:', err);
        // Return empty array to prevent page crash while allowing UI to show state
        return { users: [], smtp: null };
    }
};

export const actions: Actions = {
    registerUser: async ({ locals, request }) => {
        const fd = await request.formData();

        const display_name = String(fd.get('display_name') ?? '').trim();
        const email = String(fd.get('email') ?? '').trim().toLowerCase();
        const role = String(fd.get('role') ?? '').trim() as UserRole;

        // 1. Validation Logic
        if (!display_name || !email || !role) {
            return fail(400, {
                registerUser: { error: m.error_register_fields() },
                values: { display_name, email, role }
            });
        }

        if (!looksLikeEmail(email)) {
            return fail(400, {
                registerUser: { error: m.error_invalid_email() },
                values: { display_name, email, role }
            });
        }

        if (!ROLES.has(role)) {
            return fail(400, {
                registerUser: { error: m.error_invalid_role() },
                values: { display_name, email, role }
            });
        }

        try {
            // 2. Use SDK to create user
            const locale = String(fd.get('locale') ?? '').trim();
            const res = await locals.api.admin.createUser({
                display_name,
                email,
                role,
                locale: locale || undefined
            });

            return {
                registerUser: {
                    ok: true,
                    warning: res.warning,
                    link: res.invite_link ?? res.link
                }
            };
        } catch (err: any) {
            console.error('User registration failed:', err);
            return fail(500, {
                registerUser: { error: m.error_register_failed() },
                values: { display_name, email, role }
            });
        }
    },

    triggerPasswordReset: async ({ locals, request }) => {
        const fd = await request.formData();
        const user_id = String(fd.get('user_id') ?? '');

        if (!user_id) {
            return fail(400, { 
                triggerPasswordReset: { error: m.error_missing_user() } 
            });
        }

        try {
            // Use the new SDK function we added
            const res = await locals.api.admin.triggerPasswordReset(user_id);
            return {
                triggerPasswordReset: {
                    ok: true,
                    link: res.link,
                    message: res.message,
                    user_id
                }
            };
        } catch (err: any) {
            console.error('Trigger reset failed:', err);
            return fail(500, { 
                triggerPasswordReset: { error: m.error_reset_email_failed() } 
            });
        }
    },

    updateRole: async ({ locals, request }) => {
        const fd = await request.formData();
        const user_id = String(fd.get('user_id') ?? '');
        const new_role = String(fd.get('role') ?? '') as UserRole;

        if (!user_id || !new_role) {
            return fail(400, { updateRole: { error: m.error_role_data() } });
        }

        try {
            // Use Admin SDK to update the role
            await locals.api.admin.updateRole(user_id, new_role);
            return { updateRole: { ok: true } };
        } catch (err) {
            console.error('Role update failed:', err);
            return fail(500, { updateRole: { error: m.error_role_update() } });
        }
    },

    updateStatus: async ({ locals, request }) => {
        const fd = await request.formData();
        const user_id = String(fd.get('user_id') ?? '');
        const active = fd.get('active') === 'true';

        if (!user_id) {
            return fail(400, { updateStatus: { error: m.error_missing_user() } });
        }

        try {
            // Use Admin SDK to update status
            await locals.api.admin.updateStatus(user_id, active);
            return { updateStatus: { ok: true } };
        } catch (err: any) {
            console.error('Status update failed:', err);
            const message = err.message === 'CANNOT_DISABLE_SELF' 
                ? m.error_cannot_disable_self() 
                : m.error_status_update();
            
            return fail(400, { updateStatus: { error: message } });
        }
    }
};
