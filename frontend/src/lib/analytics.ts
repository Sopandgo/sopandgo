/*
 * Umami page analytics for the read-only public demo. The operator sets three
 * environment variables; the tracker renders only on a demo instance, so a lab
 * instance never loads analytics (docs/adr/0003).
 */

import { escapeAttribute } from './htmlAttribute';

/** The Umami variables as read from the environment, empty when unset. */
export type UmamiEnv = { scriptUrl: string; websiteId: string; domains: string };

/** A validated tracker configuration. */
export type UmamiConfig = { scriptUrl: string; websiteId: string; domains: string[] };

/**
 * What to do about Umami: nothing (null, no variable set), skip with a reason
 * to log once, or render the tracker.
 */
export type UmamiDecision = { config: UmamiConfig } | { skip: string } | null;

export function umamiConfig(demo: boolean, env: UmamiEnv): UmamiDecision {
    const scriptUrl = env.scriptUrl.trim();
    const websiteId = env.websiteId.trim();
    const domains = env.domains.trim();
    if (!scriptUrl && !websiteId && !domains) return null;
    if (!demo) return { skip: 'demo mode is off' };
    if (!isTrackerUrl(scriptUrl)) return { skip: 'UMAMI_SCRIPT_URL must be an https URL' };
    if (!UUID.test(websiteId)) return { skip: 'UMAMI_WEBSITE_ID must be a UUID' };
    const hosts = domains
        .split(',')
        .map((host) => host.trim())
        .filter((host) => host.length > 0);
    if (!hosts.every((host) => HOST_NAME.test(host))) {
        return { skip: 'UMAMI_DOMAINS must be host names separated by commas' };
    }
    return { config: { scriptUrl, websiteId, domains: hosts } };
}

/**
 * The tracker's script tag. Query strings and hashes are not counted, so SOP
 * searches and filters stay out of Umami, and Do Not Track is honoured.
 */
export function umamiScriptTag(config: UmamiConfig): string {
    const attrs = [
        `src="${escapeAttribute(config.scriptUrl)}"`,
        `data-website-id="${escapeAttribute(config.websiteId)}"`
    ];
    if (config.domains.length > 0) {
        attrs.push(`data-domains="${escapeAttribute(config.domains.join(','))}"`);
    }
    attrs.push('data-exclude-search="true"', 'data-exclude-hash="true"', 'data-do-not-track="true"');
    return `<script defer ${attrs.join(' ')}></script>`;
}

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
const HOST_NAME = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$/i;
const LOCAL_HOSTS = new Set(['localhost', '127.0.0.1']);

/** An absolute https URL, or http for a tracker on this machine. */
function isTrackerUrl(raw: string): boolean {
    let url: URL;
    try {
        url = new URL(raw);
    } catch {
        return false;
    }
    if (url.protocol === 'https:') return url.hostname !== '';
    return url.protocol === 'http:' && LOCAL_HOSTS.has(url.hostname);
}
