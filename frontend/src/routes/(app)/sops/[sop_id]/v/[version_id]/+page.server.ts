import { error, fail, redirect } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import { favorite, unfavorite } from '#lib/server/favoriteActions.js';
import { verifyAsset, verifyVersion } from '#lib/server/integrityActions.js';
import type { PageServerLoad, Actions } from './$types';
import { PDF_EXPORT_ENABLED } from '$app/env/private';

// Same limit as the backend (sop.MaxRejectReasonRunes)
const MAX_REJECT_REASON = 500;

/** Route param may be `latest`; API calls need the real version UUID. */
async function resolveVersionId(
    locals: App.Locals,
    sopId: string,
    versionParam: string
): Promise<string> {
    if (versionParam !== 'latest') return versionParam;
    const summary = await locals.api.sops.getLatestVersionSummary(sopId);
    return summary.id;
}

export const load: PageServerLoad = async ({ locals, params, parent }) => {
    const { sop } = await parent();
    const { version_id } = params;
    
    let versionSummary;

    try {
        if (version_id === 'latest') {
            versionSummary = await locals.api.sops.getLatestVersionSummary(sop.id); 
        } else {
            versionSummary = await locals.api.sops.getVersionSummary(version_id);
        }
    } catch (err) {
        console.error('Fetch Error:', err);
        
        // If 'latest' fails, redirect to the SOP root
        if (version_id === 'latest') {
            throw redirect(303, `/sops/${sop.id}`);
        }
        
        // Otherwise, it's a 404
        throw error(404, 'Version not found');
    }

    let versionDiff = null;
    try {
        versionDiff = await locals.api.sops.getVersionDiff(sop.id, versionSummary.id);
    } catch (err) {
        console.error('Version diff unavailable:', err);
    }

    const pdfExportEnabled = PDF_EXPORT_ENABLED.toLowerCase() !== 'false';

    return {
        sop,
        versionSummary,
        versionDiff,
        pdf: {
            enabled: pdfExportEnabled,
            stage: versionSummary.status,
            downloadUrl: locals.api.sops.getPdfDownloadUrl(sop.id, versionSummary.id, versionSummary.status)
        }
    };
};

export const actions: Actions = {
    verifyAsset,
    verifyVersion,

    // --- Existing Reader Acknowledgment ---
    sign: async ({ request, locals, params }) => {
        const formData = await request.formData();
        const inputName = formData.get('user_display_name')?.toString().trim();

        if (!locals.user) {
            return fail(401, { action: 'sign', error: m.error_unauthorized(), inputName });
        }

        if (inputName !== locals.user.display_name) {
            return fail(400, {
                action: 'sign',
                error: m.error_name_mismatch({ name: locals.user.display_name }),
                inputName
            });
        }

        try {
            const versionId = await resolveVersionId(locals, params.sop_id, params.version_id);
            await locals.api.sops.acknowledgeAsReader(versionId);
            return { success: true };
        } catch (err) {
            console.error('Sign Action Error:', err);
            const message = err instanceof Error ? err.message : '';
            const already = message === 'ALREADY_ACKNOWLEDGED';
            return fail(already ? 409 : 500, {
                action: 'sign',
                error: already ? m.error_already_acknowledged() : m.error_sign_failed(),
                inputName
            });
        }
    },

    // --- Lifecycle Actions ---

    promote: async ({ locals, params }) => {
        if (!locals.user) return fail(401, { action: 'promote', error: m.error_unauthorized() });

        try {
            const versionId = await resolveVersionId(locals, params.sop_id, params.version_id);
            await locals.api.sops.promoteToRC(versionId);
            return { success: true };
        } catch (err) {
            console.error('Promote Action Error:', err);
            return fail(500, { action: 'promote', error: m.error_promote_failed() });
        }
    },

    approve: async ({ request, locals, params }) => {
        if (!locals.user) return fail(401, { action: 'approve', error: m.error_unauthorized() });

        const formData = await request.formData();
        const inputName = formData.get('user_display_name')?.toString().trim();

        if (inputName !== locals.user.display_name) {
            return fail(400, {
                action: 'approve',
                error: m.error_name_mismatch({ name: locals.user.display_name }),
                inputName
            });
        }

        try {
            const versionId = await resolveVersionId(locals, params.sop_id, params.version_id);
            await locals.api.sops.approveVersion(versionId);
            return { success: true };
        } catch (err) {
            console.error('Approve Action Error:', err);
            return fail(500, { action: 'approve', error: m.error_approve_failed() });
        }
    },

    reject: async ({ request, locals, params }) => {
        if (!locals.user) return fail(401, { action: 'reject', error: m.error_unauthorized() });

        const formData = await request.formData();
        const reason = formData.get('reason')?.toString().trim();

        if (!reason) {
            return fail(400, { action: 'reject', error: m.error_reject_reason() });
        }
        if ([...reason].length > MAX_REJECT_REASON) {
            return fail(400, { action: 'reject', error: m.error_reject_reason_long({ max: String(MAX_REJECT_REASON) }), inputName: reason });
        }

        try {
            const versionId = await resolveVersionId(locals, params.sop_id, params.version_id);
            await locals.api.sops.rejectRC(versionId, reason);
            return { success: true };
        } catch (err) {
            console.error('Reject Action Error:', err);
            return fail(500, { action: 'reject', error: m.error_reject_failed(), inputName: reason });
        }
    },

    favorite,
    unfavorite
};
