import * as m from '$lib/paraglide/messages.js';
import { getLocale } from '$lib/paraglide/runtime';

const EVENT_TITLES: Record<string, () => string> = {
    login: () => m.audit_login(),
    login_failed: () => m.audit_login_failed(),
    logout: () => m.audit_logout(),
    user_created: () => m.audit_user_created(),
    user_role_updated: () => m.audit_user_role_updated(),
    user_status_changed: () => m.audit_user_status_changed(),
    user_password_updated: () => m.audit_user_password_updated(),
    admin_revoked_all_user_sessions: () => m.audit_admin_revoked_all_user_sessions(),
    system_global_revocation: () => m.audit_system_global_revocation(),
    integrity_check: () => m.audit_integrity_check(),
    system_restart: () => m.audit_system_restart(),
    sop_created: () => m.audit_sop_created(),
    sop_version_created: () => m.audit_sop_version_created(),
    sop_version_state_changed: () => m.audit_sop_version_state_changed(),
    sop_acknowledgment_created: () => m.audit_sop_acknowledgment_created(),
    acknowledgment_added: () => m.audit_acknowledgment_added(),
    asset_added: () => m.audit_asset_added(),
    tag_created: () => m.audit_tag_created(),
    tag_attached: () => m.audit_tag_attached(),
    tag_detached: () => m.audit_tag_detached(),
    tag_retired: () => m.audit_tag_retired(),
    tag_revived: () => m.audit_tag_revived(),
    email_sent: () => m.audit_email_sent(),
    email_failed: () => m.audit_email_failed(),
    backup_s3_uploaded: () => m.audit_backup_s3_uploaded(),
    default_locale_updated: () => m.audit_default_locale_updated()
};

export const auditEventOptions = Object.entries(EVENT_TITLES)
    .map(([value, label]) => ({ value, label: label() }))
    .sort((a, b) => a.label.localeCompare(b.label, getLocale()));

export const auditEntityOptions = [
    { value: 'acknowledgment', label: m.audit_entity_acknowledgment() },
    { value: 'email', label: m.audit_entity_email() },
    { value: 'sop', label: m.audit_entity_sop() },
    { value: 'sop_asset', label: m.audit_entity_sop_asset() },
    { value: 'sop_version', label: m.audit_entity_sop_version() },
    { value: 'system', label: m.audit_entity_system() },
    { value: 'tag', label: m.audit_entity_tag() },
    { value: 'user', label: m.audit_entity_user() }
];

const ENTITY_TITLES: Record<string, () => string> = {
    acknowledgment: () => m.audit_entity_acknowledgment(),
    email: () => m.audit_entity_email(),
    mail: () => m.audit_entity_email(),
    sop: () => m.audit_entity_sop(),
    sop_asset: () => m.audit_entity_sop_asset(),
    sop_version: () => m.audit_entity_sop_version(),
    system: () => m.audit_entity_system(),
    tag: () => m.audit_entity_tag(),
    user: () => m.audit_entity_user()
};

export function entityTitle(entityType: string): string {
    const known = ENTITY_TITLES[entityType.toLowerCase()];
    return known ? known() : entityType;
}

export function eventTitle(eventType: string): string {
    const key = eventType.toLowerCase();
    const known = EVENT_TITLES[key];
    if (known) return known();

    const phrase = key.replaceAll('_', ' ').trim();
    if (!phrase) return m.audit_event();
    return phrase.charAt(0).toUpperCase() + phrase.slice(1);
}

export function eventSummary(eventType: string, payload: string): string | null {
    const data = payloadObject(payload);
    if (!data) return null;

    const key = eventType.toLowerCase();
    if (key === 'login' || key === 'login_failed') {
        const ip = data.ip_address;
        if (typeof ip === 'string' && ip.length > 0) return m.audit_from({ ip: stripPort(ip) });
    }

    if (key === 'backup_s3_uploaded') {
        const objectKey = data.object_key;
        if (typeof objectKey === 'string' && objectKey.length > 0) {
            const name = objectKey.split('/').filter(Boolean).at(-1);
            if (name) return name;
        }
    }

    if (key === 'user_role_updated') {
        const from = data.from;
        const to = data.to;
        if (typeof from === 'string' && typeof to === 'string') return `${from} → ${to}`;
    }

    return null;
}

export function formatEventTime(iso: string, locale: string = getLocale()): string {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return iso;

    return new Intl.DateTimeFormat(locale, {
        day: 'numeric',
        month: 'short',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        hourCycle: 'h23'
    }).format(date);
}

export function formatPayload(payload: string): string {
    try {
        return JSON.stringify(JSON.parse(payload), null, 2);
    } catch {
        return payload;
    }
}

export function versionSopId(payload: string): string | null {
    const sopId = payloadObject(payload)?.sop_id;
    if (typeof sopId === 'string' && sopId.length > 0) return sopId;
    return null;
}

export function actorLabel(actorName: string | null): string {
    if (actorName && actorName.trim().length > 0) return actorName;
    return m.audit_actor_system();
}

function payloadObject(payload: string): Record<string, unknown> | null {
    try {
        const value = JSON.parse(payload);
        if (value && typeof value === 'object' && !Array.isArray(value)) {
            return value as Record<string, unknown>;
        }
    } catch {
        return null;
    }
    return null;
}

function stripPort(ip: string): string {
    if (ip.startsWith('[')) {
        const end = ip.indexOf(']');
        return end === -1 ? ip : ip.slice(1, end);
    }
    const colon = ip.lastIndexOf(':');
    if (colon > -1 && ip.indexOf(':') === colon) return ip.slice(0, colon);
    return ip;
}
