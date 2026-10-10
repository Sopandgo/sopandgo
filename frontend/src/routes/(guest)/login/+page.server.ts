import { fail, redirect } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { Actions } from './$types';

export const actions: Actions = {
    default: async ({ request, locals }) => {
        const data = await request.formData();
        const email = data.get('email')?.toString();
        const password = data.get('password')?.toString();

        if (!email || !password) {
            return fail(400, { message: m.error_missing_fields() });
        }

        try {
            // SDK handles API call + Cookie setting automatically
            await locals.api.auth.login(email, password);
        } catch (err) {
            const e = err as Error;
            
            if (e.message === 'INVALID_CREDENTIALS') {
                return fail(401, { message: m.error_invalid_credentials(), email });
            }
            if (e.message === 'ACCOUNT_INACTIVE') {
                return fail(403, { message: m.error_account_disabled(), email });
            }
            
            console.error('Login Error:', e);
            return fail(500, { message: m.error_system(), email });
        }

        throw redirect(303, '/dashboard');
    }
};
