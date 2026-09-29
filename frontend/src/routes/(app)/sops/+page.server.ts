import { error, fail } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, url }) => {
    try {
        // 1. Extract query params from the URL
        const limit = Number(url.searchParams.get('limit')) || 10;
        const offset = Number(url.searchParams.get('offset')) || 0;
        const q = url.searchParams.get('q') || '';
        const tag_id = url.searchParams.get('tag_id') || '';
        const favorites_only = url.searchParams.get('favorites_only') === 'true';
        const favorites_first = url.searchParams.get('favorites_first') === 'true';

        // 2. Fetch both SOPs and the list of available tags (for the filter dropdown)
        const [sopRes, tags] = await Promise.all([
            locals.api.sops.list({
                limit,
                offset,
                q,
                tag_id,
                favorites_only,
                favorites_first
            }),
            locals.api.tags.list()
        ]);

        return {
            sops: sopRes.sops,
            total: sopRes.total,
            tags,
            // Pass these back so the UI knows the current state
            filters: { limit, offset, q, tag_id, favorites_only, favorites_first }
        };
    } catch (err) {
        console.error('SOP List Error:', err);
        throw error(500, 'Could not connect to the backend');
    }
};

export const actions: Actions = {
    favorite: async ({ locals, request }) => {
        const form = await request.formData();
        const sopId = String(form.get('sop_id') ?? '').trim();
        if (!sopId) return fail(400, { message: 'Missing sop_id' });
        try {
            await locals.api.sops.favorite(sopId);
            return { success: true };
        } catch (err) {
            console.error('favorite', err);
            return fail(500, { message: 'Failed to favorite' });
        }
    },
    unfavorite: async ({ locals, request }) => {
        const form = await request.formData();
        const sopId = String(form.get('sop_id') ?? '').trim();
        if (!sopId) return fail(400, { message: 'Missing sop_id' });
        try {
            await locals.api.sops.unfavorite(sopId);
            return { success: true };
        } catch (err) {
            console.error('unfavorite', err);
            return fail(500, { message: 'Failed to unfavorite' });
        }
    }
};