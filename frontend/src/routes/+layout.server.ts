import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ locals }) => {
    // If the hook did its job, locals.user is already populated
    return {
        user: locals.user
    };
};