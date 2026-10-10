import { fail, type RequestEvent } from '@sveltejs/kit';
import * as m from '$lib/paraglide/messages.js';

/*
 * `?/favorite` and `?/unfavorite` form actions for every page that renders a
 * FavoriteToggle. Register them in the page's `actions` export.
 * The SOP comes from the form's `sop_id` (list pages) or the route's `sop_id`
 * (SOP and version pages). Results are keyed as `favorite` so they never collide
 * with other actions' `form` data; FavoriteToggle reads the error back by SOP id.
 */

type FavoriteEvent = RequestEvent<{ sop_id?: string }>;

async function setFavorite({ locals, request, params }: FavoriteEvent, on: boolean) {
    const fd = await request.formData();
    const sopId = String(fd.get('sop_id') ?? params.sop_id ?? '').trim();

    if (!sopId) return fail(400, { favorite: { sopId, error: m.error_missing_sop() } });

    try {
        if (on) await locals.api.sops.favorite(sopId);
        else await locals.api.sops.unfavorite(sopId);
        return { favorite: { sopId } };
    } catch (err) {
        console.error(on ? 'favorite' : 'unfavorite', err);
        const error = on ? m.error_favorite_failed() : m.error_unfavorite_failed();
        return fail(500, { favorite: { sopId, error } });
    }
}

export const favorite = (event: FavoriteEvent) => setFavorite(event, true);
export const unfavorite = (event: FavoriteEvent) => setFavorite(event, false);
