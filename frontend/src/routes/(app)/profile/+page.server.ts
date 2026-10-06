import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
    if (!locals.user) {
        throw redirect(303, '/login');
    }
    const signatureStatus = await locals.api.auth.getSignatureStatus().catch(() => []);

    return {
        user: locals.user,
        signatureStatus
    };
};
