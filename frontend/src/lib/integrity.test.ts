import { describe, expect, it, vi } from 'vitest';
import { SdkHttpError } from '#lib/sdk/httpError.js';
import { resolveIntegrityStatus } from './integrity';

describe('resolveIntegrityStatus', () => {
    it('maps a matching hash to verified', async () => {
        expect(await resolveIntegrityStatus(async () => ({ hash_valid: true }))).toBe('verified');
    });

    it('maps a hash mismatch to mismatch', async () => {
        expect(await resolveIntegrityStatus(async () => ({ hash_valid: false }))).toBe('mismatch');
    });

    it('maps file_missing to missing', async () => {
        const err = new SdkHttpError(404, '', { error: 'file_missing' });
        expect(await resolveIntegrityStatus(() => Promise.reject(err))).toBe('missing');
    });

    it('maps other failures to unavailable', async () => {
        vi.spyOn(console, 'error').mockImplementation(() => {});
        const notFound = new SdkHttpError(404, '', { error: 'asset_not_found' });
        const serverError = new SdkHttpError(500, '', { error: 'integrity_check_failed' });
        expect(await resolveIntegrityStatus(() => Promise.reject(notFound))).toBe('unavailable');
        expect(await resolveIntegrityStatus(() => Promise.reject(serverError))).toBe('unavailable');
        expect(await resolveIntegrityStatus(() => Promise.reject(new Error('network')))).toBe('unavailable');
    });
});
