import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ locals, url }) => {
    // 1. MANDATORY SECURITY CHECK
    // If hooks.server.ts didn't find a user, don't even talk to the Go backend.
    if (!locals.user) {
        throw error(401, 'Unauthorized');
    }

    // const sopId = url.searchParams.get('sopId');
    const sopId = "not-yet-in-use";
    const assetId = url.searchParams.get('id');
    if (!assetId) {
        throw error(400, 'Missing asset ID');
    }

    // 2. FETCH FROM GO BACKEND
    // Since locals.user exists, your SDK will now automatically 
    // attach the Authorization header.
    const res = await locals.api.fetch(`/sops/${sopId}/assets/${assetId}/download`, {
        method: 'GET'
    });

    if (!res.ok || !res.body) {
        // If Go backend says 403, we pass that through
        throw error(res.status, 'Asset not found or access denied');
    }

    const headers = new Headers();
    ['content-type', 'content-length', 'cache-control'].forEach(h => {
        const val = res.headers.get(h);
        if (val) headers.set(h, val);
    });

    // 3. STREAM THE RESPONSE
    return new Response(res.body, {
        headers: {
            'Content-Type': res.headers.get('content-type') || 'application/octet-stream',
            'Content-Length': res.headers.get('content-length') || '',
            'Cache-Control': 'private, max-age=3600' // Optional: cache for 1 hour
        }
    });
};