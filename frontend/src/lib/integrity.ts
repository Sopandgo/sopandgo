import { SdkHttpError } from '$lib/sdk/httpError';
import type { Integrity } from '$lib/sdk/types';

/**
 * Outcome of a single file integrity check.
 * - verified: file on disk matches the stored hash
 * - corrupt: file exists but its hash differs
 * - missing: record exists but the file is gone from disk
 * - unavailable: the check itself could not run (network, auth, unknown record)
 */
export type IntegrityStatus = 'verified' | 'corrupt' | 'missing' | 'unavailable';

export const INTEGRITY_STATUSES: readonly IntegrityStatus[] = ['verified', 'corrupt', 'missing', 'unavailable'];

export function isIntegrityStatus(value: unknown): value is IntegrityStatus {
    return typeof value === 'string' && (INTEGRITY_STATUSES as readonly string[]).includes(value);
}

export type IntegrityActionName = 'verifyAsset' | 'verifyVersion';

/**
 * Reads a verifyAsset / verifyVersion result for `id` out of a page's form data.
 * Used when the form posted without JavaScript and the page re-rendered.
 */
export function integrityResultFromForm(
    form: unknown,
    action: IntegrityActionName,
    id: string
): IntegrityStatus | null {
    const result = (form as Record<string, { id?: unknown; status?: unknown } | undefined> | null)?.[action];
    return result?.id === id && isIntegrityStatus(result.status) ? result.status : null;
}

/** Runs an SDK integrity call and maps its result or failure to a status. */
export async function resolveIntegrityStatus(check: () => Promise<Integrity>): Promise<IntegrityStatus> {
    try {
        const { hash_valid } = await check();
        return hash_valid ? 'verified' : 'corrupt';
    } catch (err) {
        if (err instanceof SdkHttpError && err.status === 404 && err.apiError === 'file_missing') {
            return 'missing';
        }
        console.error('Integrity check failed:', err);
        return 'unavailable';
    }
}
