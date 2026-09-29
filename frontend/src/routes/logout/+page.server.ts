import { redirect } from '@sveltejs/kit';
import type { Actions } from './$types';

export const actions: Actions = {
    default: async ({ locals }) => {
        // 1. Calls backend /auth/logout (sending the refresh token)
        // 2. Deletes access_token cookie
        // 3. Deletes refresh_token cookie
        await locals.api.auth.logout();

        throw redirect(303, '/login');
    }
};