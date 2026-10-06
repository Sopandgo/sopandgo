import { fail, redirect } from '@sveltejs/kit';
import * as m from '$lib/paraglide/messages.js';
import { isLocale } from '$lib/paraglide/runtime';
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
    setLocale: async ({ locals, request }) => {
        const fd = await request.formData();
        const locale = String(fd.get('locale') ?? '').trim();
        if (!isLocale(locale)) {
            return fail(400, { setLocale: { error: 'unsupported' } });
        }
        try {
            await locals.api.auth.updateLocale(locale);
        } catch (err) {
            console.error('Locale update failed:', err);
            return fail(400, { setLocale: { error: 'failed' } });
        }
        throw redirect(303, '/profile');
    },

    changePassword: async ({ locals, request }) => {
        const fd = await request.formData();
        const new_password = String(fd.get('new_password') ?? '');
        const confirm_password = String(fd.get('confirm_password') ?? '');

        if (new_password !== confirm_password) {
            return fail(400, { 
                changePassword: { error: m.error_passwords_mismatch() } 
            });
        }

        const user_id = locals.user?.id;
        if (!user_id) {
            return fail(401, { 
                changePassword: { error: m.error_not_authenticated() } 
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
                ? m.error_weak_password()
                : m.error_password_update_failed();

            return fail(400, { 
                changePassword: { error: msg } 
            });
        }

        // 3. Redirect to login on success
        throw redirect(303, '/login?passwordChanged=true');
    }
};