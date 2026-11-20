import { describe, expect, it } from 'vitest';
import { umamiConfig, umamiScriptTag } from './analytics';

const ID = '4f6c2a1e-8b3d-4c5e-9f70-1a2b3c4d5e6f';
const valid = { scriptUrl: 'https://cloud.umami.is/script.js', websiteId: ID, domains: '' };
const unset = { scriptUrl: '', websiteId: '', domains: '' };

describe('umamiConfig', () => {
    it('is null when no Umami variable is set, in or outside demo mode', () => {
        expect(umamiConfig(true, unset)).toBeNull();
        expect(umamiConfig(false, unset)).toBeNull();
    });

    it('renders the tracker in demo mode with valid variables', () => {
        expect(umamiConfig(true, valid)).toEqual({
            config: { scriptUrl: 'https://cloud.umami.is/script.js', websiteId: ID, domains: [] }
        });
    });

    it('never renders outside demo mode, and says why', () => {
        expect(umamiConfig(false, valid)).toEqual({ skip: 'demo mode is off' });
    });

    it('skips a script URL that is not an absolute https URL', () => {
        for (const scriptUrl of ['', 'cloud.umami.is/script.js', 'javascript:alert(1)', 'http://umami.example.org/script.js']) {
            expect(umamiConfig(true, { ...valid, scriptUrl })).toEqual({
                skip: 'UMAMI_SCRIPT_URL must be an https URL'
            });
        }
    });

    it('accepts http only for a tracker on this machine', () => {
        for (const scriptUrl of ['http://localhost:3001/script.js', 'http://127.0.0.1/script.js']) {
            expect(umamiConfig(true, { ...valid, scriptUrl })).toHaveProperty('config.scriptUrl', scriptUrl);
        }
    });

    it('skips a website ID that is not a UUID', () => {
        for (const websiteId of ['', 'abc', `${ID}x`, `"${ID}"`]) {
            expect(umamiConfig(true, { ...valid, websiteId })).toEqual({
                skip: 'UMAMI_WEBSITE_ID must be a UUID'
            });
        }
    });

    it('reads UMAMI_DOMAINS as a comma-separated list of host names', () => {
        expect(umamiConfig(true, { ...valid, domains: ' demo.sopandgo.com, ,www.example.org ' })).toHaveProperty(
            'config.domains',
            ['demo.sopandgo.com', 'www.example.org']
        );
    });

    it('skips a domain list with something other than host names', () => {
        expect(umamiConfig(true, { ...valid, domains: 'demo.sopandgo.com,"><script>' })).toEqual({
            skip: 'UMAMI_DOMAINS must be host names separated by commas'
        });
    });
});

describe('umamiScriptTag', () => {
    it('loads the tracker without query strings, hashes or Do Not Track visitors', () => {
        expect(
            umamiScriptTag({ scriptUrl: 'https://cloud.umami.is/script.js', websiteId: ID, domains: [] })
        ).toBe(
            `<script defer src="https://cloud.umami.is/script.js" data-website-id="${ID}" ` +
                'data-exclude-search="true" data-exclude-hash="true" data-do-not-track="true"></script>'
        );
    });

    it('limits counting to the listed domains', () => {
        expect(
            umamiScriptTag({
                scriptUrl: 'https://cloud.umami.is/script.js',
                websiteId: ID,
                domains: ['demo.sopandgo.com', 'www.example.org']
            })
        ).toContain(' data-domains="demo.sopandgo.com,www.example.org" ');
    });

    it('escapes attribute values', () => {
        const tag = umamiScriptTag({
            scriptUrl: 'https://umami.example.org/s.js?a=1&b="2"<x>',
            websiteId: ID,
            domains: []
        });
        expect(tag).toContain('src="https://umami.example.org/s.js?a=1&amp;b=&quot;2&quot;&lt;x&gt;"');
    });
});
