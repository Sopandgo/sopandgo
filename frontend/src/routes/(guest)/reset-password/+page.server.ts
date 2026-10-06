import { fail, redirect } from '@sveltejs/kit';
import * as m from '$lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';

// 1. Load the token from URL so the page can put it in the form
export const load: PageServerLoad = async ({ url }) => {
    const token = url.searchParams.get('token');
    if (!token) {
        // If they arrive without a token, kick them out or show an error state
        throw redirect(303, '/login?error=missing_token');
    }
    return { token };
};

export const actions: Actions = {
    default: async ({ request, locals }) => {
        const data = await request.formData();
        const token = data.get('token')?.toString();
        const password = data.get('password')?.toString();

        if (!token || !password) {
            return fail(400, { message: m.error_missing_fields() });
        }

        try {
            await locals.api.auth.resetPassword(token, password);
        } catch (err) {
            const e = err as Error;
            
            if (e.message === 'TOKEN_INVALID_OR_EXPIRED') {
                return fail(401, { message: m.error_invite_expired() });
            }
            
            return fail(400, { message: e.message });
        }

        throw redirect(303, '/login?reset=success');
    }
};