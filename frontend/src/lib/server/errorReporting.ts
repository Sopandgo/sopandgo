/*
 * Reports the SvelteKit server's unexpected errors to Sentry or GlitchTip,
 * with the configuration an admin saved (GET /api/error-reporting/config,
 * cached for a minute). Built on @sentry/core alone: @sentry/node would bring
 * OpenTelemetry, and @sentry/sveltekit a build-time plugin.
 */

import { BACKEND_URL } from '$app/env/private';
import {
    Client,
    Scope,
    createTransport,
    eventFromMessage,
    eventFromUnknownInput,
    resolvedSyncPromise,
    type BaseTransportOptions,
    type ClientOptions,
    type Event,
    type EventHint,
    type ParameterizedString,
    type SeverityLevel,
    type StackParser
} from '@sentry/core';
import {
    browserHeadTag,
    createConfigCache,
    NO_DATA_COLLECTION,
    nodeStackParser,
    scrubEvent,
    type ErrorReportingConfig
} from '#lib/errorReporting.js';

const stackParser: StackParser = (stack, skipFirstLines = 0) => nodeStackParser(stack, skipFirstLines);

class ServerClient extends Client<ClientOptions> {
    constructor(options: ClientOptions) {
        super(options);
    }

    eventFromException(exception: unknown, hint?: EventHint): PromiseLike<Event> {
        return resolvedSyncPromise(eventFromUnknownInput(this, stackParser, exception, hint));
    }

    eventFromMessage(message: ParameterizedString, level: SeverityLevel = 'info', hint?: EventHint): PromiseLike<Event> {
        return resolvedSyncPromise(eventFromMessage(stackParser, message, level, hint, true));
    }
}

/** Sends envelopes with the runtime's fetch. */
function makeFetchTransport(options: BaseTransportOptions) {
    return createTransport(options, async (request) => {
        const res = await fetch(options.url, {
            method: 'POST',
            body: request.body as BodyInit,
            headers: options.headers
        });
        return {
            statusCode: res.status,
            headers: {
                'x-sentry-rate-limits': res.headers.get('X-Sentry-Rate-Limits'),
                'retry-after': res.headers.get('Retry-After')
            }
        };
    });
}

const configCache = createConfigCache(
    async () => {
        const base = BACKEND_URL.replace(/\/$/, '');
        const res = await fetch(`${base}/api/error-reporting/config`);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return (await res.json()) as ErrorReportingConfig;
    },
    {
        ttlMs: 60_000,
        onError: (err) =>
            console.warn('Error reporting configuration unavailable; not reporting:', (err as Error).message)
    }
);

let client: ServerClient | null = null;
let clientKey = '';

/** The client for the current configuration; replaced when an admin changes it. */
async function currentClient(): Promise<ServerClient | null> {
    const config = await configCache.get();
    const key = config.enabled && config.dsn ? JSON.stringify([config.dsn, config.environment, config.release]) : '';
    if (key !== clientKey) {
        void client?.close(2000);
        client = key ? newClient(config) : null;
        clientKey = key;
    }
    return client;
}

function newClient(config: ErrorReportingConfig): ServerClient {
    const c = new ServerClient({
        dsn: config.dsn,
        environment: config.environment || undefined,
        release: config.release || undefined,
        transport: makeFetchTransport,
        stackParser,
        integrations: [],
        dataCollection: NO_DATA_COLLECTION,
        maxBreadcrumbs: 0,
        beforeSend: (event) => scrubEvent(event)
    });
    c.init();
    return c;
}

/**
 * Reports an unexpected server error. Never throws: handleError must not.
 * Tags name the route ID, never the URL.
 */
export async function reportServerError(error: unknown, tags: Record<string, string>): Promise<void> {
    try {
        const c = await currentClient();
        if (!c) return;
        const scope = new Scope();
        scope.setClient(c);
        scope.setTags({ ...tags, runtime: 'sveltekit-server' });
        c.captureException(error, undefined, scope);
    } catch (err) {
        console.warn('Error reporting failed:', (err as Error).message);
    }
}

/** The browser reporting tag for %sopandgo.head%, or '' while it is off. */
export async function browserReportingHead(): Promise<string> {
    try {
        return browserHeadTag(await configCache.get());
    } catch {
        return '';
    }
}
