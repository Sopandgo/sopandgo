import type { RequestEvent } from '@sveltejs/kit';

/**
 * The visitor's address, which the SDK forwards to the API on every call for
 * its per-address rate limits. Caddy sets X-Real-IP to the visitor it
 * resolved ({client_ip}: the peer, or the visitor behind a proxy listed in
 * TRUSTED_PROXIES) and overwrites any value the client sent.
 */
export function visitorAddress(event: Pick<RequestEvent, 'request' | 'getClientAddress'>): string {
    const fromProxy = event.request.headers.get('x-real-ip')?.trim();
    if (fromProxy) return fromProxy;
    try {
        return event.getClientAddress();
    } catch {
        return '';
    }
}
