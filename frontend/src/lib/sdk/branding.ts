import type { Client } from './client';
import type { Branding } from './types';

export const branding = (client: Client) => ({
    /** Public: the sign-in page needs it, so no token is sent. */
    get: async (): Promise<Branding> => {
        const res = await client.public('/branding', { method: 'GET' });
        if (!res.ok) throw new Error('BRANDING_FETCH_FAILED');
        return await res.json();
    }
});
