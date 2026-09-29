import { error, fail, redirect } from '@sveltejs/kit';
import type { PageServerLoad, Actions } from './$types';
import { env } from '$env/dynamic/private';

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

    const pdfExportEnabled = (env.PDF_EXPORT_ENABLED ?? 'true').toLowerCase() !== 'false';

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
    // --- Existing Reader Acknowledgment ---
    sign: async ({ request, locals, params }) => {
        const formData = await request.formData();
        const inputName = formData.get('user_display_name')?.toString().trim();
        const versionId = params.version_id;

        if (!locals.user) {
            return fail(401, { error: 'Unauthorized', inputName });
        }

        if (inputName !== locals.user.display_name) {
            return fail(400, { 
                error: `Name verification failed. Please enter "${locals.user.display_name}" exactly.`,
                inputName
            });
        }

        try {
            await locals.api.sops.acknowledgeAsReader(versionId);
            return { success: true };
        } catch (err) {
            console.error('Sign Action Error:', err);
            return fail(500, { 
                error: 'Failed to sign the document. Please try again.',
                inputName 
            });
        }
    },

    // --- Lifecycle Actions ---

    promote: async ({ locals, params }) => {
        if (!locals.user) return fail(401, { error: 'Unauthorized' });

        try {
            await locals.api.sops.promoteToRC(params.version_id);
            return { success: true };
        } catch (err) {
            console.error('Promote Action Error:', err);
            return fail(500, { error: 'Failed to promote to Release Candidate.' });
        }
    },

    approve: async ({ request, locals, params }) => {
        if (!locals.user) return fail(401, { error: 'Unauthorized' });

        const formData = await request.formData();
        const inputName = formData.get('user_display_name')?.toString().trim();
        const versionId = params.version_id;

        if (!locals.user) {
            return fail(401, { error: 'Unauthorized', inputName });
        }

        if (inputName !== locals.user.display_name) {
            return fail(400, { 
                error: `Name verification failed. Please enter "${locals.user.display_name}" exactly.`,
                inputName
            });
        }

        try {
            await locals.api.sops.approveVersion(params.version_id);
            return { success: true };
        } catch (err) {
            console.error('Approve Action Error:', err);
            return fail(500, { error: 'Failed to approve and publish SOP.' });
        }
    },

    reject: async ({ request, locals, params }) => {
        if (!locals.user) return fail(401, { error: 'Unauthorized' });

        const formData = await request.formData();
        const reason = formData.get('reason')?.toString().trim();

        if (!reason) {
            return fail(400, { error: 'A reason must be provided to reject an SOP.' });
        }

        try {
            await locals.api.sops.rejectRC(params.version_id, reason);
            return { success: true };
        } catch (err) {
            console.error('Reject Action Error:', err);
            return fail(500, { error: 'Failed to reject the Release Candidate.' });
        }
    },

    favorite: async ({ locals, params }) => {
        const sopId = params.sop_id;
        if (!sopId) return fail(400, { error: 'Missing sop_id' });
        try {
            await locals.api.sops.favorite(sopId);
            return { success: true };
        } catch (err) {
            console.error('favorite', err);
            return fail(500, { error: 'Failed to favorite' });
        }
    },

    unfavorite: async ({ locals, params }) => {
        const sopId = params.sop_id;
        if (!sopId) return fail(400, { error: 'Missing sop_id' });
        try {
            await locals.api.sops.unfavorite(sopId);
            return { success: true };
        } catch (err) {
            console.error('unfavorite', err);
            return fail(500, { error: 'Failed to unfavorite' });
        }
    }
};