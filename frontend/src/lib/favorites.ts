/**
 * Reads the error of a favorite / unfavorite action for `sopId` out of a page's
 * form data (see $lib/server/favoriteActions). Null when the last favorite
 * action was for another SOP, succeeded, or never ran.
 */
export function favoriteErrorFromForm(form: unknown, sopId: string): string | null {
    const result = (form as { favorite?: { sopId?: unknown; error?: unknown } } | null)?.favorite;
    return result?.sopId === sopId && typeof result.error === 'string' ? result.error : null;
}
