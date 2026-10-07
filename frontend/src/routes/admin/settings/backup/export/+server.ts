import { BACKEND_URL } from '$env/static/private';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ fetch, cookies }) => {
    const token = cookies.get('access_token');
    if (!token) throw error(401, 'Unauthorized');

    const base = BACKEND_URL.replace(/\/$/, '');
    const res = await fetch(`${base}/api/admin/backups/export`, {
        method: 'GET',
        headers: {
            Authorization: `Bearer ${token}`
        }
    });

    if (!res.ok || !res.body) {
        const msg = await res.text();
        throw error(res.status || 500, msg || 'Failed to export backup');
    }

    return new Response(res.body, {
        status: 200,
        headers: {
            'content-type': res.headers.get('content-type') ?? 'application/zip',
            'content-disposition':
                res.headers.get('content-disposition') ?? 'attachment; filename="sopandgo-backup.zip"'
        }
    });
};
