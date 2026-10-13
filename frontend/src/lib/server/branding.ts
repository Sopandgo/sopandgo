import { DEFAULT_BRANDING } from '#lib/branding.js';
import type { Branding } from '#lib/sdk/types.js';

/*
 * Every page shows the branding, so it is cached in this process instead of
 * asking the backend on each request. The branding tab clears the cache after
 * a save; the TTL only matters if the backend is changed some other way.
 */
const TTL_MS = 60_000;

let cached: { value: Branding; expires: number } | null = null;

export async function getBranding(api: App.Locals['api']): Promise<Branding> {
	const now = Date.now();
	if (cached && cached.expires > now) return cached.value;
	try {
		const value = await api.branding.get();
		cached = { value, expires: now + TTL_MS };
		return value;
	} catch (err) {
		// The sign-in page must still render; retry on the next request.
		console.error('Failed to load branding:', err);
		return cached?.value ?? DEFAULT_BRANDING;
	}
}

export function invalidateBranding(): void {
	cached = null;
}
