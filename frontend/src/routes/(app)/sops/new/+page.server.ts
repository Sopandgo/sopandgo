import { fail, redirect } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import type { Actions } from './$types';

export const actions: Actions = {
    default: async ({ request, locals }) => {
        const formData = await request.formData();
        const title = formData.get('title')?.toString();

        if (!title) {
            return fail(400, { message: m.error_title_required() });
        }

        try {
            const { id } = await locals.api.sops.create(title);

            throw redirect(303, `/sops/${id}/new`);

        } catch (err) {
            if ((err as { status?: number }).status === 303) throw err;

            console.error('Create SOP Error:', err);
            
            return fail(500, { 
                message: m.error_create_sop(), 
                title 
            });
        }
    }
};
