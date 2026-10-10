import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

// Same-origin proxy for profile pictures: the browser cannot send the httpOnly
// access token to the Go backend, so the image goes through locals.api.

export const GET: RequestHandler = async ({ locals, url, request }) => {
    if (!locals.user) {
        throw error(401, 'Unauthorized');
    }

    const userId = url.searchParams.get('user_id');
    if (!userId) {
        throw error(400, 'Missing user ID');
    }

    // The `v` parameter (avatar content hash) only busts the browser cache; it is not forwarded.
    const size = url.searchParams.get('size') === 'md' ? 'md' : 'sm';

    const headers: Record<string, string> = {};
    const ifNoneMatch = request.headers.get('if-none-match');
    if (ifNoneMatch) headers['If-None-Match'] = ifNoneMatch;

    const res = await locals.api.fetch(`/users/${encodeURIComponent(userId)}/avatar?size=${size}`, {
        method: 'GET',
        headers
    });

    if (res.status === 304) {
        return new Response(null, {
            status: 304,
            headers: {
                ...(res.headers.get('etag') ? { ETag: res.headers.get('etag')! } : {}),
                'Cache-Control': res.headers.get('cache-control') || 'private, max-age=3600'
            }
        });
    }

    if (!res.ok || !res.body) {
        throw error(res.status === 404 ? 404 : res.status >= 500 ? 502 : res.status, 'Avatar not available');
    }

    const out = new Headers();
    out.set('Content-Type', res.headers.get('content-type') || 'image/jpeg');
    const etag = res.headers.get('etag');
    if (etag) out.set('ETag', etag);
    out.set('Cache-Control', res.headers.get('cache-control') || 'private, max-age=3600');

    return new Response(res.body, { headers: out });
};
