import { describe, expect, it } from 'vitest';
import { visitorAddress } from './visitorAddress';

function event(headers: Record<string, string>, clientAddress: () => string) {
    return { request: new Request('http://localhost/', { headers }), getClientAddress: clientAddress };
}

describe('visitorAddress', () => {
    it('uses the address Caddy resolved', () => {
        expect(visitorAddress(event({ 'x-real-ip': ' 203.0.113.9 ' }, () => '127.0.0.1'))).toBe('203.0.113.9');
    });

    it('falls back to the peer without Caddy', () => {
        expect(visitorAddress(event({}, () => '::1'))).toBe('::1');
    });

    it('returns nothing when the peer is unknown', () => {
        expect(
            visitorAddress(
                event({}, () => {
                    throw new Error('no address');
                })
            )
        ).toBe('');
    });
});
