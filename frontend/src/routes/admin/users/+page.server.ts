import { fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import type { UserRole } from '$lib/sdk/types';

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
                registerUser: { error: 'Missing required fields.' },
                values: { display_name, email, role }
            });
        }

        if (!looksLikeEmail(email)) {
            return fail(400, {
                registerUser: { error: 'Invalid email address.' },
                values: { display_name, email, role }
            });
        }

        if (!ROLES.has(role)) {
            return fail(400, {
                registerUser: { error: 'Invalid role.' },
                values: { display_name, email, role }
            });
        }

        try {
            // 2. Use SDK to create user
            const res = await locals.api.admin.createUser({
                display_name,
                email,
                role,
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
                registerUser: { error: 'Failed to register user. Email may already be in use.' },
                values: { display_name, email, role }
            });
        }
    },

    triggerPasswordReset: async ({ locals, request }) => {
        const fd = await request.formData();
        const user_id = String(fd.get('user_id') ?? '');

        if (!user_id) {
            return fail(400, { 
                triggerPasswordReset: { error: 'Missing user ID.' } 
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
                triggerPasswordReset: { error: 'Failed to send reset email.' } 
            });
        }
    },

    updateRole: async ({ locals, request }) => {
        const fd = await request.formData();
        const user_id = String(fd.get('user_id') ?? '');
        const new_role = String(fd.get('role') ?? '') as UserRole;

        if (!user_id || !new_role) {
            return fail(400, { updateRole: { error: 'Invalid user or role data.' } });
        }

        try {
            // Use Admin SDK to update the role
            await locals.api.admin.updateRole(user_id, new_role);
            return { updateRole: { ok: true } };
        } catch (err) {
            console.error('Role update failed:', err);
            return fail(500, { updateRole: { error: 'Failed to update user role.' } });
        }
    },

    updateStatus: async ({ locals, request }) => {
        const fd = await request.formData();
        const user_id = String(fd.get('user_id') ?? '');
        const active = fd.get('active') === 'true';

        if (!user_id) {
            return fail(400, { updateStatus: { error: 'Missing user ID.' } });
        }

        try {
            // Use Admin SDK to update status
            await locals.api.admin.updateStatus(user_id, active);
            return { updateStatus: { ok: true } };
        } catch (err: any) {
            console.error('Status update failed:', err);
            const message = err.message === 'CANNOT_DISABLE_SELF' 
                ? 'You cannot disable your own account.' 
                : 'Failed to update user status.';
            
            return fail(400, { updateStatus: { error: message } });
        }
    }
};