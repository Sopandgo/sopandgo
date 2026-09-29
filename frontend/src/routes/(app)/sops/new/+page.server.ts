import { fail, redirect } from '@sveltejs/kit';
import type { Actions } from './$types';

export const actions: Actions = {
    default: async ({ request, locals }) => {
        const formData = await request.formData();
        const title = formData.get('title')?.toString();

        if (!title) {
            return fail(400, { message: 'Title is required' });
        }

        try {
            const { id } = await locals.api.sops.create(title);

            throw redirect(303, `/sops/${id}`);

        } catch (err) {
            if ((err as { status?: number }).status === 303) throw err;

            console.error('Create SOP Error:', err);
            
            return fail(500, { 
                message: 'Failed to create SOP. Please try again.', 
                title 
            });
        }
    }
};