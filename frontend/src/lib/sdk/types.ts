export type UserRole = 'admin' | 'approver' | 'editor' | 'viewer' | 'auditor';
export type ThemePreference = 'light' | 'dark' | 'system';

export interface User {
    /** UUID v4 */
    id: string;
    display_name: string;
    email: string;
    role: UserRole;
    is_active: boolean;
    /** When true, the user must change their password before using the app */
    must_change_password: boolean;
    /** BCP 47 tag from the supported locale list */
    locale: string;
    /** light, dark, or system */
    theme: ThemePreference;
    /** ISO 8601 Timestamp */
    created_at: string;
}
    
export interface Acknowledgment {
    user_id: string;
    display_name: string;
    ack_type: string;
    created_at: string;
}

export interface AcknowledgmentWithUser extends Acknowledgment {
    user: User;
}

// 1. Define the specific failure shapes
export interface AdminSOPIntegrityFailure {
    sop_id: string;
    version?: number;
    asset_id?: string;
    error: string;
}

export interface AdminAuditIntegrityFailure {
    event_id: number;
    error: string;
}

// 2. Define the section shapes
export interface AdminSOPIntegritySection {
    checked: number;
    valid: number;
    invalid: number;
    failed?: AdminSOPIntegrityFailure[]; 
}

export interface AdminAuditIntegritySection {
    checked: number;
    valid: number;
    invalid: number;
    failed?: AdminAuditIntegrityFailure[]; 
}

// 3. Update the main payload to include the audit section
export interface AdminIntegrity {
    ok: boolean;
    checked_at: string;
    audit: AdminAuditIntegritySection;
    versions: AdminSOPIntegritySection;
    assets: AdminSOPIntegritySection;
}

export interface SOPAsset {
    id: string;
    file_name: string;
    content_hash: string;
    created_at: string;
}

export interface Integrity {
    hash_valid: boolean;
}

export interface SOP {
    id: string;
    title: string;
    created_at: string;
    tags: Tag[];
    latest_published_version_id: string | null;
    /** Present on list/detail when the backend includes per-user favorite state */
    is_favorite?: boolean;
}

export interface Tag {
    /** UUID v4 */
    id: string;
    title: string;
    is_active: boolean;
}

// Add this near your other SOP interfaces
export interface SOPListResponse {
    sops: SOP[];
    total: number;
}

export interface SOPHistory {
    id: string;
    title: string;
    versions: SOPVersion[];
}

export type SOPVersionStatus = 'draft' | 'rc' | 'published' | 'rejected' | 'superseded';

export interface SOPVersion {
    id: string;
    version: number;
    created_at: string;
    content_hash: string;
    change_summary: string;
    status: SOPVersionStatus;
}

export interface SOPVersionSummary {
    id: string;
    sop_id: string;
    title: string;
    version: number;
    created_at: string;
    change_summary: string;
    content: string;
    content_hash: string;
    hash_valid: boolean;
    status: SOPVersionStatus;
    assets: SOPAsset[];
    acknowledgments: AcknowledgmentWithUser[];
    tags: Tag[];
}

export interface AuditEvent {
    id: string;
    event_type: string;
    entity_type: string;
    entity_id: string;
    actor_user_id: string | null;
    actor_name: string | null;
    payload: string;
    /** ISO 8601 timestamp */
    created_at: string;
    hash: string;
    prev_hash: string;
    hash_valid: boolean;
}

export interface AuditLogResponse {
    events: AuditEvent[];
    total: number;
}

export interface AuditFilterOptionsResponse {
    event_types: string[];
    entity_types: string[];
}

/** Admin GET /admin/settings/email (no secrets); same shape as legacy /admin/settings/smtp */
export interface PublicSmtpSettings {
    host: string;
    port: string;
    username: string;
    from_address: string;
    mail_mode: 'smtp' | 'manual_links';
    /** What invites and resets actually use: email only when it is on and the chosen transport is saved. */
    effective_mail_mode: 'smtp' | 'manual_links';
    mail_transport: 'smtp' | 'resend';
    default_locale: string;
    password_configured: boolean;
    configured: boolean;
    encryption_key_set: boolean;
    resend_from_address: string;
    resend_configured: boolean;
    resend_api_key_configured: boolean;
}

export type IntegrationEvent =
    | 'sop_published'
    | 'sop_rc'
    | 'sop_rejected'
    | 'backup_s3_failed'
    | 'integrity_check_failed';

export interface PublicIntegrationChannel {
    enabled: boolean;
    events: IntegrationEvent[];
    configured: boolean;
    secret_configured: boolean;
    url?: string;
}

export interface PublicIntegrationSettings {
    encryption_key_set: boolean;
    slack: PublicIntegrationChannel;
    gotify: PublicIntegrationChannel;
    webhook: PublicIntegrationChannel;
    known_events: IntegrationEvent[];
}

export interface AdminCreateUserResponse {
    id: string;
    warning?: string;
    message?: string;
    invite_link?: string;
    link?: string;
}

export interface AdminPasswordResetResponse {
    message: string;
    link?: string;
}

export interface BackupManifest {
    backup_format_version: number;
    app_version: string;
    created_at_utc: string;
    db_schema_version: number;
    data_layout_version: number;
}

/** Saved scheduled S3 backup settings (Settings → Backup). The secret is never returned. */
export interface BackupS3Settings {
    encryption_key_set: boolean;
    enabled: boolean;
    /** A bucket is saved. */
    configured: boolean;
    bucket: string;
    region: string;
    key_prefix: string;
    endpoint: string;
    use_path_style: boolean;
    /** Empty = the server's default AWS credential chain. */
    access_key_id: string;
    secret_configured: boolean;
    /** Go duration, e.g. `24h0m0s`. */
    interval: string;
    retention_max: number;
    retention_days: number;
}

/** PUT body: an empty `secret_access_key` keeps the stored secret. */
export interface BackupS3SettingsInput {
    enabled: boolean;
    bucket: string;
    region: string;
    key_prefix: string;
    endpoint: string;
    use_path_style: boolean;
    access_key_id: string;
    secret_access_key: string;
    interval: string;
    retention_max: number;
    retention_days: number;
}

/** Admin backup status: automatic S3 uploads (when switched on under Settings → Backup). */
export interface S3ScheduledBackupStatus {
    enabled: boolean;
    bucket?: string;
    key_prefix?: string;
    interval?: string;
    retention_max_objects?: number;
    retention_days?: number;
    last_run_utc?: string;
    last_success_utc?: string;
    last_object_key?: string;
    last_error?: string;
    next_run_utc?: string;
}

export interface BackupStatus {
    locked: boolean;
    message: string;
    pending_restore: boolean;
    s3_scheduled?: S3ScheduledBackupStatus;
}

export interface BackupImportValidation {
    ok: boolean;
    manifest: BackupManifest;
    note: string;
}

export interface BackupApplyResponse {
    ok: boolean;
    manifest: BackupManifest;
    requires_restart: boolean;
    pre_apply_backup: string;
    note: string;
}

export interface DiffLine {
    kind: 'context' | 'add' | 'del';
    text: string;
}

export interface VersionDiff {
    comparable: boolean;
    from_version_id?: string;
    from_version?: number;
    to_version_id: string;
    to_version: number;
    lines: DiffLine[];
}

export interface PublishedActivity {
    sop_id: string;
    title: string;
    version_id: string;
    version: number;
    change_summary: string;
    published_at: string;
    published_by: string;
}

export interface TrainingMember {
    user_id: string;
    display_name: string;
    signed_at?: string;
}

export interface SOPTrainingCoverage {
    sop_id: string;
    title: string;
    has_published: boolean;
    version_id?: string;
    version?: number;
    signed: TrainingMember[];
    unsigned: TrainingMember[];
}

export interface UserSignatureStatus {
    sop_id: string;
    title: string;
    latest_version_id: string;
    latest_version: number;
    has_signed_latest: boolean;
    signed_older_version: boolean;
    last_sign_date?: string;
}