import { fail, redirect } from '@sveltejs/kit';
import type { Actions } from './$types';

export const actions: Actions = {
    default: async ({ request, locals }) => {
        const data = await request.formData();
        const email = data.get('email')?.toString();
        const password = data.get('password')?.toString();

        if (!email || !password) {
            return fail(400, { message: 'Missing fields' });
        }

        try {
            // SDK handles API call + Cookie setting automatically
            await locals.api.auth.login(email, password);
        } catch (err) {
            const e = err as Error;
            
            if (e.message === 'INVALID_CREDENTIALS') {
                return fail(401, { message: 'Invalid credentials.', email });
            }
            if (e.message === 'ACCOUNT_INACTIVE') {
                return fail(403, { message: 'Your account is disabled.', email });
            }
            
            console.error('Login Error:', e);
            return fail(500, { message: 'System error.', email });
        }

        throw redirect(303, '/dashboard');
    }
};