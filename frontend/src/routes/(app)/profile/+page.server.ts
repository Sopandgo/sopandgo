import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
    // Ensure the user is authenticated (handled by hooks usually, but safe to double check)
    if (!locals.user) {
        throw redirect(303, '/login');
    }
    // Fetch signature status
    const signatureStatus = await locals.api.auth.getSignatureStatus().catch(() => []);

    return {
        user: locals.user,
        signatureStatus
    };
};

export const actions: Actions = {
    changePassword: async ({ locals, request }) => {
        const fd = await request.formData();
        const new_password = String(fd.get('new_password') ?? '');
        const confirm_password = String(fd.get('confirm_password') ?? '');

        if (new_password !== confirm_password) {
            return fail(400, { 
                changePassword: { error: 'Passwords do not match.' } 
            });
        }

        const user_id = locals.user?.id;
        if (!user_id) {
            return fail(401, { 
                changePassword: { error: 'Not authenticated.' } 
            });
        }

        try {
            // 1. Use SDK to update password
            await locals.api.auth.updatePassword(new_password);

            // 2. Use SDK to logout (This handles API call AND cookie deletion)
            await locals.api.auth.logout();

        } catch (err: any) {
            console.error('Change Password Error:', err);
            
            // Map SDK errors to user messages
            const msg = err.message === 'WEAK_PASSWORD' 
                ? 'Password must be at least 12 characters and include at least 3 of: lowercase, uppercase, number, and symbol.'
                : 'Failed to update password. Please try again.';

            return fail(400, { 
                changePassword: { error: msg } 
            });
        }

        // 3. Redirect to login on success
        throw redirect(303, '/login?passwordChanged=true');
    }
};