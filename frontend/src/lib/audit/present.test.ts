import { describe, expect, it } from 'vitest';
import * as m from '#lib/paraglide/messages.js';
import { eventSummary, eventTitle, formatEventTime } from './present';

describe('audit presentation', () => {
    it('uses the catalog for known event titles', () => {
        expect(eventTitle('login')).toBe(m.audit_login());
        expect(m.audit_login({}, { locale: 'de' })).toBe('Angemeldet');
    });

    it('labels branding events and shows a name change', () => {
        expect(eventTitle('branding_reset')).toBe(m.audit_branding_reset());
        expect(eventSummary('branding_updated', JSON.stringify({ old_name: '', new_name: 'Lab Handbook' }))).toBe(
            'SOPandGO → Lab Handbook'
        );
        expect(eventSummary('branding_updated', JSON.stringify({ old_name: 'Lab', new_name: 'Lab' }))).toBeNull();
    });

    it('formats the event time in the active locale', () => {
        const iso = '2026-10-06T12:00:00Z';
        expect(formatEventTime(iso, 'en').toLowerCase()).toContain('oct');
        expect(formatEventTime(iso, 'de').toLowerCase()).toContain('okt');
    });
});
