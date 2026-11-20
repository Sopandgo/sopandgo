import type { RequestEvent } from '@sveltejs/kit';

/**
 * The visitor's address, for the demo session rate limit in the API. Caddy
 * sets X-Real-IP to the peer it saw and overwrites any value the client sent.
 */
export function visitorAddress(event: RequestEvent): string {
    const fromProxy = event.request.headers.get('x-real-ip')?.trim();
    if (fromProxy) return fromProxy;
    try {
        return event.getClientAddress();
    } catch {
        return '';
    }
}
