import { json, error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ locals, request }) => {
    // Security Check: Only authenticated users can trigger integrity checks
    if (!locals.user) {
        throw error(401, 'Unauthorized');
    }

    const { assetId } = await request.json();
    const sopId = "not-in-use-yet";

    if (!assetId) {
        return json({ error: 'Missing assetId' }, { status: 400 });
    }

    try {
        // Use the new SDK method
        const { hash_valid } = await locals.api.assets.checkIntegrity(sopId, assetId);
        
        return json({ success: hash_valid });
    } catch (err) {
        console.error('Integrity check proxy failed:', err);
        // We return success: false rather than a 500 error to let the UI 
        // handle the "Corrupt/Missing" state gracefully.
        return json({ success: false });
    }
};