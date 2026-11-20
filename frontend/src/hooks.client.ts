import type { ClientInit, HandleClientError } from '@sveltejs/kit/hooks';
import { BROWSER_META_NAME, readBrowserConfig, shouldReport } from '#lib/errorReporting.js';
import type { Report } from '#lib/browserErrorReporting.js';

/*
 * Browser error reporting, only when an admin turned it on: the server then
 * puts a meta tag into the page (see $lib/errorReporting). Without it, no
 * Sentry code is loaded. Loading does not hold up the app's start.
 */

let ready: Promise<Report | null> = Promise.resolve(null);

export const init: ClientInit = () => {
    const meta = document.querySelector(`meta[name="${BROWSER_META_NAME}"]`);
    const config = readBrowserConfig(meta?.getAttribute('content'));
    if (!config) return;
    ready = import('#lib/browserErrorReporting.js')
        .then(({ startBrowserReporting }) => startBrowserReporting(config))
        .catch(() => null);
};

export const handleError: HandleClientError = ({ error, kind, event }) => {
    if (!shouldReport(kind)) return;
    const tags = { route: event.route.id ?? '(unmatched)', runtime: 'browser' };
    void ready.then((report) => report?.(error, tags));
};
