import { fail } from '@sveltejs/kit';
import { invalidateBranding } from '#lib/server/branding.js';
import * as m from '#lib/paraglide/messages.js';
import type { Actions } from './$types';

/** Matches the backend limit (backend/internal/branding). */
const MAX_LOGO_BYTES = 2 * 1024 * 1024;

function errorCode(err: unknown): string {
    return err instanceof Error ? err.message : '';
}

export const actions: Actions = {
    saveText: async ({ locals, request }) => {
        const fd = await request.formData();
        const input = {
            name: String(fd.get('name') ?? '').trim(),
            primary_light: String(fd.get('primary_light') ?? '').trim(),
            primary_dark: String(fd.get('primary_dark') ?? '').trim()
        };
        try {
            await locals.api.admin.updateBranding(input);
        } catch (err) {
            const code = errorCode(err);
            const error =
                code === 'NAME_INVALID'
                    ? m.branding_name_invalid()
                    : code === 'COLOUR_INVALID'
                      ? m.branding_colour_invalid()
                      : code === 'COLOUR_CONTRAST'
                        ? m.branding_colour_contrast()
                        : m.branding_save_failed();
            if (!code.startsWith('NAME_') && !code.startsWith('COLOUR_')) {
                console.error('Branding update failed:', err);
            }
            return fail(400, { saveText: { error, input } });
        } finally {
            invalidateBranding();
        }
        return { saveText: { ok: true } };
    },

    uploadLogo: async ({ locals, request }) => {
        const fd = await request.formData();
        const file = fd.get('file');
        if (!(file instanceof File) || file.size === 0) {
            return fail(400, { logo: { error: m.branding_logo_missing() } });
        }
        if (file.size > MAX_LOGO_BYTES) {
            return fail(400, { logo: { error: m.branding_logo_too_large() } });
        }
        try {
            await locals.api.admin.uploadBrandingLogo(file);
        } catch (err) {
            const code = errorCode(err);
            const error =
                code === 'INVALID_IMAGE'
                    ? m.branding_logo_invalid()
                    : ['IMAGE_TOO_LARGE', 'FILE_TOO_LARGE', 'MULTIPART_INVALID'].includes(code)
                      ? m.branding_logo_too_large()
                      : m.branding_save_failed();
            if (error === m.branding_save_failed()) console.error('Logo upload failed:', err);
            return fail(400, { logo: { error } });
        } finally {
            invalidateBranding();
        }
        return { logo: { ok: true } };
    },

    reset: async ({ locals }) => {
        try {
            await locals.api.admin.resetBranding();
        } catch (err) {
            console.error('Branding reset failed:', err);
            return fail(500, { reset: { error: m.branding_save_failed() } });
        } finally {
            invalidateBranding();
        }
        return { reset: { ok: true } };
    },

    removeLogo: async ({ locals }) => {
        try {
            await locals.api.admin.removeBrandingLogo();
        } catch (err) {
            // Already gone (e.g. removed in another tab) is fine.
            if (errorCode(err) !== 'NO_LOGO') {
                console.error('Logo removal failed:', err);
                return fail(500, { logo: { error: m.branding_save_failed() } });
            }
        } finally {
            invalidateBranding();
        }
        return { logo: { ok: true } };
    }
};
