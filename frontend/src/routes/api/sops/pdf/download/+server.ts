import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ locals, url }) => {
    if (!locals.user) {
        throw error(401, 'Unauthorized');
    }

    const sopId = url.searchParams.get('sopId');
    const versionId = url.searchParams.get('id');
    const stage = url.searchParams.get('stage');

    if (!sopId || !versionId) {
        throw error(400, 'Missing sopId or version id');
    }

    const query = new URLSearchParams();
    if (stage) {
        query.set('stage', stage);
    }

    const apiPath = `/sops/${sopId}/versions/${versionId}/download-pdf` + (query.toString() ? `?${query.toString()}` : '');
    const res = await locals.api.fetch(apiPath, {
        method: 'GET'
    });

    if (!res.ok || !res.body) {
        throw error(res.status, 'PDF artifact not found or access denied');
    }

    const headers = new Headers();
    ['content-type', 'content-length', 'content-disposition', 'cache-control'].forEach((h) => {
        const val = res.headers.get(h);
        if (val) headers.set(h, val);
    });

    return new Response(res.body, { headers });
};
