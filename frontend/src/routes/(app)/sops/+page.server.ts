import { error } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import { favorite, unfavorite } from '#lib/server/favoriteActions.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, url }) => {
    try {
        // 1. Extract query params from the URL
        const limit = Number(url.searchParams.get('limit')) || 25;
        const offset = Number(url.searchParams.get('offset')) || 0;
        const q = url.searchParams.get('q') || '';
        const tag_id = url.searchParams.get('tag_id') || '';
        const favorites_only = url.searchParams.get('favorites_only') === 'true';
        const favorites_first = url.searchParams.get('favorites_first') === 'true';
        // The default list is SOPs in use; archived=only shows the archived ones
        const archived = url.searchParams.get('archived') === 'only';

        // 2. Fetch both SOPs and the list of available tags (for the filter dropdown)
        const [sopRes, tags] = await Promise.all([
            locals.api.sops.list({
                limit,
                offset,
                q,
                tag_id,
                favorites_only,
                favorites_first,
                archived: archived ? 'only' : undefined
            }),
            locals.api.tags.list()
        ]);

        // With nothing in use and no filter set, find out whether everything is archived,
        // so the page can still offer the way to the archived SOPs
        const unfiltered = !q && !tag_id && !favorites_only;
        const archivedTotal =
            !archived && unfiltered && sopRes.total === 0
                ? (await locals.api.sops.list({ limit: 1, offset: 0, archived: 'only' })).total
                : 0;

        return {
            sops: sopRes.sops,
            total: sopRes.total,
            archivedTotal,
            tags,
            // Pass these back so the UI knows the current state
            filters: { limit, offset, q, tag_id, favorites_only, favorites_first, archived }
        };
    } catch (err) {
        console.error('SOP List Error:', err);
        throw error(500, m.error_service_unavailable());
    }
};

export const actions: Actions = {
    favorite,
    unfavorite
};
