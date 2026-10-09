import { fail, type RequestEvent } from '@sveltejs/kit';
import { resolveIntegrityStatus } from '$lib/integrity';

/*
 * Form actions shared by every page under /sops/[sop_id] that lists assets or
 * shows a version. Register them in the page's `actions` export.
 * Results are keyed so they never collide with other actions' `form` data.
 */

export async function verifyAsset({ locals, request, params }: RequestEvent<{ sop_id: string }>) {
    const fd = await request.formData();
    const id = String(fd.get('asset_id') ?? '').trim();
    const sopId = params.sop_id;

    if (!sopId || !id) return fail(400, { verifyAsset: { id, status: 'unavailable' as const } });

    const status = await resolveIntegrityStatus(() => locals.api.assets.checkIntegrity(sopId, id));
    return { verifyAsset: { id, status } };
}

export async function verifyVersion({ locals, request, params }: RequestEvent<{ sop_id: string }>) {
    const fd = await request.formData();
    const id = String(fd.get('version_id') ?? '').trim();
    const sopId = params.sop_id;

    if (!sopId || !id) return fail(400, { verifyVersion: { id, status: 'unavailable' as const } });

    const status = await resolveIntegrityStatus(() => locals.api.sops.checkIntegrity(sopId, id));
    return { verifyVersion: { id, status } };
}
