import { describe, expect, it } from 'vitest';
import * as m from '$lib/paraglide/messages.js';
import { eventTitle, formatEventTime } from './present';

describe('audit presentation', () => {
    it('uses the catalog for known event titles', () => {
        expect(eventTitle('login')).toBe(m.audit_login());
        expect(m.audit_login({}, { locale: 'de' })).toBe('Angemeldet');
    });

    it('formats the event time in the active locale', () => {
        const iso = '2026-10-06T12:00:00Z';
        expect(formatEventTime(iso, 'en').toLowerCase()).toContain('oct');
        expect(formatEventTime(iso, 'de').toLowerCase()).toContain('okt');
    });
});
