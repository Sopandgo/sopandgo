import { getBranding } from '#lib/server/branding.js';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ locals }) => {
    // If the hook did its job, locals.user is already populated
    return {
        user: locals.user,
        demo: locals.demo,
        branding: await getBranding(locals.api)
    };
};