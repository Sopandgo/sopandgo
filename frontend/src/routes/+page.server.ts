import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

// No landing page: `/` only sends people to where they belong.
export const load: PageServerLoad = async ({ locals }) => {
    // A demo visitor lands on the SOP library.
    if (locals.user && locals.demo) throw redirect(303, '/sops');
    throw redirect(303, locals.user ? '/dashboard' : '/login');
};
