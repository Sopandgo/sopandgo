/*
 * Browser error reporting, loaded by hooks.client.ts only when an admin
 * turned it on. Named imports keep the bundle to the client, the fetch
 * transport and two integrations: no replay, tracing or breadcrumbs.
 */

import {
    BrowserClient,
    dedupeIntegration,
    defaultStackParser,
    getCurrentScope,
    globalHandlersIntegration,
    makeFetchTransport
} from '@sentry/browser';
import { NO_DATA_COLLECTION, scrubEvent, type BrowserReportingConfig } from './errorReporting';

export type Report = (error: unknown, tags: Record<string, string>) => void;

/** Starts the client; the returned function reports one error with tags. */
export function startBrowserReporting(config: BrowserReportingConfig): Report {
    const client = new BrowserClient({
        dsn: config.dsn,
        environment: config.environment || undefined,
        release: config.release || undefined,
        transport: makeFetchTransport,
        stackParser: defaultStackParser,
        // Uncaught errors and unhandled rejections, without duplicates.
        integrations: [globalHandlersIntegration(), dedupeIntegration()],
        dataCollection: NO_DATA_COLLECTION,
        maxBreadcrumbs: 0,
        beforeSend: (event) => scrubEvent(event)
    });
    getCurrentScope().setClient(client);
    client.init();
    return (error, tags) => {
        const scope = getCurrentScope().clone();
        scope.setTags(tags);
        client.captureException(error, undefined, scope);
    };
}
