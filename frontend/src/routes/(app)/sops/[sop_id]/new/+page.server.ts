import { fail, redirect } from '@sveltejs/kit';
import type { Actions, PageServerLoad } from './$types';
import type { SOPAsset } from '$lib/sdk/types';
import {
	messageForAssetUploadFailure,
	messageForPublishFailure
} from '$lib/utils/actionErrorMessages';

export const load: PageServerLoad = async ({ parent, locals }) => {
    const { sop } = await parent();
    
    let assets: SOPAsset[] = [];
    try {
        // Fetch assets specifically for this SOP container
        assets = await locals.api.assets.list(sop.id);
    } catch (e) {
        console.error("Failed to load assets:", e);
        assets = [];
    }

    return { 
        sop,
        assets 
    };
};

export const actions: Actions = {
    // Action to Publish the Version
    publish: async ({ request, locals, params }) => {
        const formData = await request.formData();
        const content = formData.get('content')?.toString();
        const changeSummary = formData.get('change_summary')?.toString() ?? '';
        const sopId = params.sop_id;

        if (!content || content.trim().length === 0) {
            return fail(400, { message: 'Content is required', content, change_summary: changeSummary });
        }
        if (!changeSummary.trim()) {
            return fail(400, {
                message: 'Describe what changed and why.',
                content,
                change_summary: changeSummary
            });
        }

        try {
            const res = await locals.api.sops.createVersion(sopId, content, changeSummary.trim());
            throw redirect(303, `/sops/${sopId}/v/${res.id}`);
        } catch (err) {
            if ((err as { status?: number }).status === 303) throw err;
            console.error('Create Version Error:', err);
            const message = messageForPublishFailure(err);
            const status =
                typeof err === 'object' &&
                err !== null &&
                'status' in err &&
                typeof (err as { status: number }).status === 'number'
                    ? (err as { status: number }).status
                    : 500;
            const code = status >= 400 && status < 500 ? status : 500;
            return fail(code, { message, content, change_summary: changeSummary });
        }
    },

    // Action to Upload an Asset (called from the sidebar)
    upload: async ({ request, locals, params }) => {
        const formData = await request.formData();
        const file = formData.get('file') as File;
        const sopId = params.sop_id;

        if (!file || file.size === 0) {
            return fail(400, { uploadError: 'Please select a file to upload.' });
        }

        try {
            // The SDK expects (file, sopId)
            await locals.api.assets.upload(file, sopId);
            
            // We return success. SvelteKit will automatically re-run 'load',
            // refreshing the asset list in the sidebar.
            return { uploadSuccess: true };

        } catch (err) {
            console.error('Upload Error:', err);
            const uploadError = messageForAssetUploadFailure(err);
            const status =
                typeof err === 'object' &&
                err !== null &&
                'status' in err &&
                typeof (err as { status: number }).status === 'number'
                    ? (err as { status: number }).status
                    : 500;
            const code = status >= 400 && status < 500 ? status : 500;
            return fail(code, { uploadError });
        }
    }
};