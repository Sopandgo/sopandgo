const EVENT_TITLES: Record<string, string> = {
    login: 'Signed in',
    login_failed: 'Sign-in failed',
    logout: 'Signed out',
    user_created: 'User created',
    user_role_updated: 'Role changed',
    user_status_changed: 'User status changed',
    user_password_updated: 'Password updated',
    admin_revoked_all_user_sessions: 'Sessions revoked',
    system_global_revocation: 'All sessions revoked',
    integrity_check: 'Integrity checked',
    system_restart: 'System restarted',
    sop_created: 'SOP created',
    sop_version_created: 'SOP version created',
    sop_version_state_changed: 'SOP version updated',
    sop_acknowledgment_created: 'SOP acknowledgment recorded',
    acknowledgment_added: 'Acknowledgment added',
    asset_added: 'Asset added',
    tag_created: 'Tag created',
    tag_attached: 'Tag attached',
    tag_detached: 'Tag detached',
    tag_retired: 'Tag retired',
    tag_revived: 'Tag revived',
    email_sent: 'Email sent',
    email_failed: 'Email failed',
    backup_s3_uploaded: 'Backup uploaded'
};

export const auditEventOptions = Object.entries(EVENT_TITLES)
    .map(([value, label]) => ({ value, label }))
    .sort((a, b) => a.label.localeCompare(b.label));

export const auditEntityOptions = [
    { value: 'acknowledgment', label: 'Acknowledgment' },
    { value: 'email', label: 'Email' },
    { value: 'sop', label: 'SOP' },
    { value: 'sop_asset', label: 'SOP asset' },
    { value: 'sop_version', label: 'SOP version' },
    { value: 'system', label: 'System' },
    { value: 'tag', label: 'Tag' },
    { value: 'user', label: 'User' }
];

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

export function eventTitle(eventType: string): string {
    const key = eventType.toLowerCase();
    const known = EVENT_TITLES[key];
    if (known) return known;

    const phrase = key.replaceAll('_', ' ').trim();
    if (!phrase) return 'Event';
    return phrase.charAt(0).toUpperCase() + phrase.slice(1);
}

export function eventSummary(eventType: string, payload: string): string | null {
    const data = payloadObject(payload);
    if (!data) return null;

    const key = eventType.toLowerCase();
    if (key === 'login' || key === 'login_failed') {
        const ip = data.ip_address;
        if (typeof ip === 'string' && ip.length > 0) return `from ${stripPort(ip)}`;
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

export function formatEventTime(iso: string): string {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return iso;

    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    return `${date.getDate()} ${MONTHS[date.getMonth()]} ${date.getFullYear()}, ${hours}:${minutes}`;
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
    return 'System';
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

function stripPort(address: string): string {
    const ipv4 = address.match(/^(\d{1,3}(?:\.\d{1,3}){3}):\d+$/);
    if (ipv4) return ipv4[1];

    const ipv6 = address.match(/^\[(.+)\]:\d+$/);
    if (ipv6) return ipv6[1];

    return address;
}
