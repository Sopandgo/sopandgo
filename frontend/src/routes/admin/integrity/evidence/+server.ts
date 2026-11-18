import { BACKEND_URL } from '$app/env/private';
import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

// Streams the audit evidence bundle (audit log and anchor receipts) from the backend.
export const GET: RequestHandler = async ({ fetch, cookies }) => {
    const token = cookies.get('access_token');
    if (!token) throw error(401, 'Unauthorized');

    const base = BACKEND_URL.replace(/\/$/, '');
    const res = await fetch(`${base}/api/admin/audit-anchoring/bundle`, {
        method: 'GET',
        headers: { Authorization: `Bearer ${token}` }
    });

    if (!res.ok || !res.body) {
        const msg = await res.text();
        throw error(res.status || 500, msg || 'Failed to download the evidence bundle');
    }

    return new Response(res.body, {
        status: 200,
        headers: {
            'content-type': res.headers.get('content-type') ?? 'application/zip',
            'content-disposition':
                res.headers.get('content-disposition') ?? 'attachment; filename="audit-evidence.zip"',
            'cache-control': 'no-store'
        }
    });
};
