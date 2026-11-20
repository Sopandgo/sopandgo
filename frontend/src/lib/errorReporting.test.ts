import { describe, expect, it } from 'vitest';
import {
    browserHeadTag,
    createConfigCache,
    nodeStackParser,
    readBrowserConfig,
    scrubEvent,
    shouldReport,
    type ErrorReportingConfig
} from './errorReporting';

const DSN = 'https://abc@glitchtip.example.org/7';
const on = { enabled: true, dsn: DSN, environment: 'lab-a', release: '1.3.0', browser: true };

describe('browserHeadTag', () => {
    it('is empty unless reporting and browser reporting are both on', () => {
        expect(browserHeadTag(null)).toBe('');
        expect(browserHeadTag({ enabled: false })).toBe('');
        expect(browserHeadTag({ ...on, browser: false })).toBe('');
        expect(browserHeadTag({ ...on, dsn: '' })).toBe('');
    });

    it('hands browsers the DSN, environment and release, and nothing they could execute', () => {
        const tag = browserHeadTag({ ...on, environment: 'x"><script>alert(1)</script>' });
        expect(tag.startsWith('<meta name="sopandgo-error-reporting" content="')).toBe(true);
        expect(tag).not.toContain('<script');
        expect(tag).not.toContain('"><');
        const content = tag.match(/content="([^"]*)"/)?.[1] ?? '';
        const decoded = content.replace(/&quot;/g, '"').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&amp;/g, '&');
        expect(readBrowserConfig(decoded)).toEqual({
            dsn: DSN,
            environment: 'x"><script>alert(1)</script>',
            release: '1.3.0'
        });
    });
});

describe('readBrowserConfig', () => {
    it('is null without a usable meta tag', () => {
        for (const content of [null, undefined, '', 'not json', '{}', '{"dsn":""}', '{"dsn":42}']) {
            expect(readBrowserConfig(content)).toBeNull();
        }
    });
});

describe('shouldReport', () => {
    it('reports only unexpected errors, not error() responses, 404s or invalid input', () => {
        expect(shouldReport('unknown')).toBe(true);
        for (const kind of ['app', 'framework', 'validation']) expect(shouldReport(kind)).toBe(false);
    });
});

describe('scrubEvent', () => {
    it('drops the request, the user, breadcrumbs and contexts that are not about the runtime', () => {
        const event = scrubEvent({
            message: 'boom',
            tags: { route: '/(app)/sops/[id]' },
            request: { url: 'https://lab.example.org/sops/42?q=secret', headers: { cookie: 'refresh_token=x' } },
            user: { id: 'u1', ip_address: '203.0.113.7', email: 'a@lab.example.org' },
            breadcrumbs: [{ message: 'clicked Sign' }],
            contexts: { os: { name: 'Linux' }, browser: { name: 'Firefox' }, form_data: { title: 'secret' } }
        });
        expect(event).toEqual({
            message: 'boom',
            tags: { route: '/(app)/sops/[id]' },
            contexts: { os: { name: 'Linux' }, browser: { name: 'Firefox' } }
        });
    });

    it('strips query strings and fragments from stack frame files and the transaction', () => {
        const event = scrubEvent({
            transaction: '/sops/42?tab=history#v2',
            exception: {
                values: [
                    {
                        type: 'TypeError',
                        stacktrace: {
                            frames: [
                                { filename: 'https://lab.example.org/_app/immutable/app.js?v=1#x', lineno: 3 },
                                { filename: 'file:///app/frontend-build/server/index.js', lineno: 9 },
                                { function: 'anonymous' }
                            ]
                        }
                    }
                ]
            }
        });
        expect(event.transaction).toBe('/sops/42');
        expect(event.exception?.values?.[0]?.stacktrace?.frames?.map((f) => f.filename)).toEqual([
            'https://lab.example.org/_app/immutable/app.js',
            'file:///app/frontend-build/server/index.js',
            undefined
        ]);
    });
});

describe('createConfigCache', () => {
    function setup() {
        let clock = 0;
        const answers: Array<ErrorReportingConfig | Error> = [];
        let loads = 0;
        const errors: unknown[] = [];
        const cache = createConfigCache(
            async () => {
                loads++;
                const next = answers.shift() ?? { enabled: false };
                if (next instanceof Error) throw next;
                return next;
            },
            { ttlMs: 60_000, now: () => clock, onError: (err) => errors.push(err) }
        );
        return {
            cache,
            answers,
            errors,
            loads: () => loads,
            advance: (ms: number) => (clock += ms)
        };
    }

    it('asks the API once a minute, so a change in the admin UI arrives within a minute', async () => {
        const t = setup();
        t.answers.push(on, { enabled: false });
        expect(await t.cache.get()).toEqual(on);
        t.advance(59_999);
        expect(await t.cache.get()).toEqual(on);
        expect(t.loads()).toBe(1);
        t.advance(1);
        expect(await t.cache.get()).toEqual({ enabled: false });
        expect(t.loads()).toBe(2);
    });

    it('shares one request between callers that ask at the same time', async () => {
        const t = setup();
        t.answers.push(on);
        const [a, b] = await Promise.all([t.cache.get(), t.cache.get()]);
        expect(a).toEqual(on);
        expect(b).toEqual(on);
        expect(t.loads()).toBe(1);
    });

    it('counts an unreachable API as off, and logs once per run of failures', async () => {
        const t = setup();
        t.answers.push(new Error('down'), new Error('still down'), on, new Error('down again'));
        expect(await t.cache.get()).toEqual({ enabled: false });
        t.advance(60_000);
        expect(await t.cache.get()).toEqual({ enabled: false });
        expect(t.errors).toHaveLength(1);
        t.advance(60_000);
        expect(await t.cache.get()).toEqual(on);
        t.advance(60_000);
        await t.cache.get();
        expect(t.errors).toHaveLength(2);
    });
});

describe('nodeStackParser', () => {
    it('reads V8 stack lines into frames, oldest first, and marks Node internals as not ours', () => {
        const stack = [
            'TypeError: boom',
            '    at load (file:///app/frontend-build/server/chunks/page.js:12:7)',
            '    at async Promise.all (index 0)',
            '    at /app/frontend-build/server/index.js:3:9',
            '    at Module._compile (node:internal/modules/cjs/loader:1554:14)',
            '    at render (/app/node_modules/svelte/internal.js:40:2)'
        ].join('\n');
        expect(nodeStackParser(stack)).toEqual([
            { filename: '/app/node_modules/svelte/internal.js', function: 'render', lineno: 40, colno: 2, in_app: false },
            { filename: 'node:internal/modules/cjs/loader', function: 'Module._compile', lineno: 1554, colno: 14, in_app: false },
            { filename: '/app/frontend-build/server/index.js', function: '?', lineno: 3, colno: 9, in_app: true },
            { filename: 'file:///app/frontend-build/server/chunks/page.js', function: 'load', lineno: 12, colno: 7, in_app: true }
        ]);
    });
});
