import { error } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ locals, params }) => {
    // Note: Ensure your folder is named [sop_id], not [id]
    const { sop_id } = params;

    try {
        // 1. Fetch SOP Metadata (Shared across all tabs/subpages)
        const sop = await locals.api.sops.getById(sop_id);
        
        return { sop };
    } catch (err) {
        console.error('SOP Layout Error:', err);
        throw error(404, 'SOP not found');
    }
};