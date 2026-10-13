import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

// Same-origin proxy for the organization logo. Public on purpose: the sign-in
// page shows it. The backend only serves PNGs it re-encoded itself.

export const GET: RequestHandler = async ({ locals, url, request }) => {
    // The `v` parameter (logo hash) only busts the browser cache; it is not forwarded.
    const variant = url.searchParams.get('variant') === 'icon' ? 'icon' : 'logo';

    const headers: Record<string, string> = {};
    const ifNoneMatch = request.headers.get('if-none-match');
    if (ifNoneMatch) headers['If-None-Match'] = ifNoneMatch;

    const res = await locals.api.public(`/branding/logo?variant=${variant}`, { method: 'GET', headers });

    if (res.status === 304) {
        const etag = res.headers.get('etag');
        return new Response(null, {
            status: 304,
            headers: {
                ...(etag ? { ETag: etag } : {}),
                'Cache-Control': res.headers.get('cache-control') || 'public, max-age=300'
            }
        });
    }
    if (!res.ok || !res.body) {
        throw error(res.status === 404 ? 404 : 502, 'Logo not available');
    }

    const out = new Headers();
    out.set('Content-Type', 'image/png');
    out.set('X-Content-Type-Options', 'nosniff');
    const etag = res.headers.get('etag');
    if (etag) out.set('ETag', etag);
    out.set('Cache-Control', res.headers.get('cache-control') || 'public, max-age=300');

    return new Response(res.body, { headers: out });
};
