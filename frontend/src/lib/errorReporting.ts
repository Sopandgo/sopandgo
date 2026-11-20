/*
 * Error reporting to Sentry or GlitchTip, shared by the SvelteKit server and
 * browsers. An admin turns it on under Settings → Error reporting; the Go API
 * publishes the configuration at GET /api/error-reporting/config. Nothing in
 * this module talks to the network.
 */

import type { DataCollection, Event } from '@sentry/core';
import { escapeAttribute } from './htmlAttribute';

/** GET /api/error-reporting/config. The DSN is only present while on. */
export type ErrorReportingConfig = {
    enabled: boolean;
    dsn?: string;
    environment?: string;
    release?: string;
    browser?: boolean;
};

/** What browsers need to report. */
export type BrowserReportingConfig = { dsn: string; environment: string; release: string };

/** The meta tag name hooks.client.ts reads. */
export const BROWSER_META_NAME = 'sopandgo-error-reporting';

/**
 * The tag for %sopandgo.head% that turns browser reporting on, or '' when
 * reporting or browser reporting is off.
 */
export function browserHeadTag(config: ErrorReportingConfig | null): string {
    if (!config?.enabled || !config.browser || !config.dsn) return '';
    const content: BrowserReportingConfig = {
        dsn: config.dsn,
        environment: config.environment ?? '',
        release: config.release ?? ''
    };
    return `<meta name="${BROWSER_META_NAME}" content="${escapeAttribute(JSON.stringify(content))}">`;
}

/** Parses the meta tag's content; null when absent or unusable. */
export function readBrowserConfig(content: string | null | undefined): BrowserReportingConfig | null {
    if (!content) return null;
    let parsed: unknown;
    try {
        parsed = JSON.parse(content);
    } catch {
        return null;
    }
    if (typeof parsed !== 'object' || parsed === null) return null;
    const { dsn, environment, release } = parsed as Record<string, unknown>;
    if (typeof dsn !== 'string' || dsn === '') return null;
    return {
        dsn,
        environment: typeof environment === 'string' ? environment : '',
        release: typeof release === 'string' ? release : ''
    };
}

/**
 * What the Sentry SDKs may collect on their own: nothing about users,
 * cookies, headers, bodies, query strings or local variables. scrubEvent
 * removes the same again before anything is sent.
 */
export const NO_DATA_COLLECTION: DataCollection = {
    userInfo: false,
    cookies: false,
    httpHeaders: false,
    httpBodies: [],
    urlQueryParams: false,
    stackFrameVariables: false
};

/**
 * Whether SvelteKit's handleError should report an error. Only unexpected
 * errors: error() responses, 404s and invalid input are the app working.
 */
export function shouldReport(kind: string): boolean {
    return kind === 'unknown';
}

/** The only contexts sent: what the runtime, OS and browser are. */
const KEPT_CONTEXTS = new Set(['os', 'runtime', 'device', 'browser', 'trace']);

/**
 * The last step before an event leaves, on the server and in browsers: no
 * request, user or breadcrumbs, only runtime contexts, and no query strings
 * or fragments in file names or the transaction (docs/adr/0004).
 */
export function scrubEvent<E extends Event>(event: E): E {
    delete event.request;
    delete event.user;
    delete event.breadcrumbs;
    if (event.contexts) {
        for (const name of Object.keys(event.contexts)) {
            if (!KEPT_CONTEXTS.has(name)) delete event.contexts[name];
        }
    }
    if (event.transaction) event.transaction = withoutQuery(event.transaction);
    for (const value of event.exception?.values ?? []) {
        for (const frame of value.stacktrace?.frames ?? []) {
            if (frame.filename) frame.filename = withoutQuery(frame.filename);
            if (frame.abs_path) frame.abs_path = withoutQuery(frame.abs_path);
        }
    }
    return event;
}

function withoutQuery(value: string): string {
    return value.replace(/[?#].*$/, '');
}

const OFF: ErrorReportingConfig = { enabled: false };

/**
 * Caches the configuration for ttlMs. Callers that ask at the same time share
 * one request. A failed request counts as off; onError hears about the first
 * failure of each run, so an API that is down does not flood the log.
 */
export function createConfigCache(
    load: () => Promise<ErrorReportingConfig>,
    options: { ttlMs: number; now?: () => number; onError?: (err: unknown) => void }
) {
    const now = options.now ?? Date.now;
    let value: ErrorReportingConfig = OFF;
    let fetchedAt = -Infinity;
    let inFlight: Promise<ErrorReportingConfig> | null = null;
    let failing = false;

    async function refresh(): Promise<ErrorReportingConfig> {
        try {
            value = await load();
            failing = false;
        } catch (err) {
            value = OFF;
            if (!failing) options.onError?.(err);
            failing = true;
        }
        fetchedAt = now();
        return value;
    }

    return {
        get(): Promise<ErrorReportingConfig> {
            if (now() - fetchedAt < options.ttlMs) return Promise.resolve(value);
            inFlight ??= refresh().finally(() => {
                inFlight = null;
            });
            return inFlight;
        }
    };
}

/** A stack frame as Sentry expects it. */
export type StackFrame = {
    filename: string;
    function: string;
    lineno: number;
    colno: number;
    in_app: boolean;
};

// "    at fn (file:line:col)" or "    at file:line:col"
const V8_FRAME = /^\s*at (?:(.+?) \((.+):(\d+):(\d+)\)|(.+):(\d+):(\d+))$/;
const MAX_FRAMES = 50;

/**
 * Parses a V8 (Node) stack into frames, oldest first as Sentry expects. Lines
 * without a file position, such as "at async Promise.all (index 0)", are
 * skipped. Node internals and node_modules are not the app's code.
 * @sentry/core has no Node parser, and @sentry/node would bring OpenTelemetry.
 */
export function nodeStackParser(stack: string, skipFirstLines = 0): StackFrame[] {
    const frames: StackFrame[] = [];
    for (const line of stack.split('\n').slice(skipFirstLines)) {
        const match = V8_FRAME.exec(line);
        if (!match) continue;
        const filename = match[2] ?? match[5];
        frames.push({
            filename,
            function: match[1] ?? '?',
            lineno: Number(match[3] ?? match[6]),
            colno: Number(match[4] ?? match[7]),
            in_app: !filename.startsWith('node:') && !filename.includes('/node_modules/')
        });
        if (frames.length === MAX_FRAMES) break;
    }
    return frames.reverse();
}
