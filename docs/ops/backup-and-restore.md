# Backup and Restore

This document describes how to back up and restore sopandgo installations.

It is intended for administrators and technical staff responsible for operating
the system.

## Overview

sopandgo is designed to make backup and restore **simple and reliable**.

All persistent state is stored in a single **data directory** (`DATA_DIR` in the
process environment; in Docker Compose this is usually mounted as `./backend/data`
on the host and `/app/data` inside the container).

No application data is stored in external services.

## What needs to be backed up

The data directory contains:

- the SQLite database (`app.db`) — metadata, audit events, acknowledgments, encrypted SMTP and integration settings
- SOP content — Markdown version files and assets under `sops/`

The SMTP password and integration secrets are stored encrypted; recovery still requires the same
`SMTP_SECRET_ENCRYPTION_KEY` (or re-entered secrets after key rotation) when
`mail_mode` is `smtp` or outbound integrations are enabled.
If `mail_mode` is `manual_links`, invites/resets do not require SMTP delivery.

If the full data directory is backed up, the entire system state is preserved.

---

## Admin UI backup (recommended for operators)

Sign in as **admin** and open **Settings → Backup**: `/admin/settings/backup`. `/admin/backup` redirects there.

### Export

- Downloads a `.zip` containing:
  - `app.db` — consistent SQLite snapshot (via `VACUUM INTO` while a short maintenance lock is held)
  - `sops/` — SOP files on disk (if present)
  - `manifest.json` — format version, optional `APP_VERSION`, schema version, timestamp

Export is also available via the API: `GET /api/admin/backups/export` (admin auth required).

### Validate

- Upload a `.zip` to check structure and compatibility **without** changing live data.
- Rejects backups whose **database schema is newer** than the running app (older-or-equal schema is allowed; migrations run on startup after restore).

### Apply (staged restore)

- Destructive: stages replacement of live data from the uploaded archive.
- Requires typing **`APPLY BACKUP`** exactly as confirmation.
- Before staging, the server writes a **pre-apply** snapshot zip under:
  - `data/_backup_snapshots/` (filename prefix `pre-apply-`)
- Staged payload is written to:
  - `data/_restore_pending/`
- **The restore is not active until the next backend process start.** Restart the
  container or process (e.g. `docker compose restart`) so startup can apply
  `_restore_pending` and then run normal migrations.
- While a restore is pending (`_restore_pending` present), further validate/apply
  uploads are blocked until after restart clears it.

Environment variable (optional):

- **`APP_VERSION`** — recorded in export `manifest.json` for traceability. Release images bake this in at build time. Labs do not set it. It is `dev` when the process is started without that value.

---

## Automatic S3 backups (optional)

When enabled, the backend uploads the **same** `.zip` as manual export on a fixed
interval. Objects are stored under:

`<BACKUP_S3_PREFIX>/automated/sopandgo-automated-YYYYMMDDThhmmss.zip`

(`BACKUP_S3_PREFIX` is optional; if set, leading/trailing slashes are trimmed.)

| Variable | Meaning |
|----------|---------|
| **`BACKUP_S3_ENABLED`** | Set to `true` / `1` / `yes` to enable uploads. |
| **`BACKUP_S3_BUCKET`** | Target bucket name (required when enabled). |
| **`BACKUP_S3_REGION`** | AWS region for the SDK (set for AWS; for MinIO you can use a placeholder such as `us-east-1` if needed). |
| **`BACKUP_S3_PREFIX`** | Optional key prefix (e.g. `prod/myorg`). |
| **`BACKUP_S3_INTERVAL`** | Go duration between runs (default **`24h`**). Minimum **`1m`**. |
| **`BACKUP_S3_RETENTION_MAX`** | After each successful upload, delete older automated archives until at most this many **non-expired** objects remain (`0` = no count limit). |
| **`BACKUP_S3_RETENTION_DAYS`** | Delete automated archives whose S3 `LastModified` is older than this many days (`0` = no age limit). |
| **`BACKUP_S3_ENDPOINT`** | Optional custom endpoint (MinIO, LocalStack, etc.). |
| **`BACKUP_S3_USE_PATH_STYLE`** | Set `true` for path-style addressing (common with MinIO). |
| **`BACKUP_S3_ACCESS_KEY_ID`** / **`BACKUP_S3_SECRET_ACCESS_KEY`** | Optional static credentials. If **both** are omitted, the AWS SDK default chain is used (container / instance role, env vars, shared config). |

**Retention semantics:** objects matching the automated naming pattern under the
prefix are listed. Any object **older than** `BACKUP_S3_RETENTION_DAYS` (when set)
is deleted. Among those that remain, if `BACKUP_S3_RETENTION_MAX` is set and the
count is still greater than the limit, the **oldest** extras are deleted until
the count is at most the limit.

The admin **Backup** page and `GET /api/admin/backups/status` include a
`s3_scheduled` object (bucket, prefix, interval, retention settings, last
success, next run, last error) for operators. **Secrets are never returned.**

Successful uploads and failures are also written to the **audit log**
(`backup_s3_uploaded`, `backup_s3_failed`).

**Security notes:** use a dedicated IAM principal with least privilege on the
bucket/prefix; prefer **versioning** and (for stronger ransomware resistance)
**S3 Object Lock** or a separate backup account. Manual export/download is
unchanged and does not require S3 configuration.

---

## Recommended backup strategy

### General principles

- Backups should be performed **regularly**
- Backups should be stored **outside the host system**
- Restore procedures should be **tested periodically**

Use **automatic S3 backups** (above), **manual export**, **API export**, or
filesystem copies depending on your environment.

### Simple filesystem backup (classic)

The safest manual method is to **stop the container** before copying data.

1. Stop the sopandgo container
2. Copy the entire data directory to a backup location
3. Restart the container

This guarantees a fully consistent backup of `app.db` and WAL files.

Example (conceptual):

```sh
docker stop sopandgo
cp -r /path/to/data /path/to/backup/sopandgo-YYYY-MM-DD
docker start sopandgo
```

### Online backup (advanced)

For environments where downtime is not acceptable, online backups are possible.

sopandgo uses SQLite in WAL (write-ahead logging) mode. Copying only `app.db`
while the app runs is **not** sufficient; use SQLite backup APIs or rely on the
admin **export** path, which snapshots via `VACUUM INTO`.

If in doubt, prefer stopping the container or using **Export** from the admin UI.

### Backup frequency

Typical recommendations:

- Small labs: daily or weekly
- Active environments: daily
- Before upgrades: always

The appropriate frequency depends on how often SOPs are updated.

---

## Restore procedure

### From admin staged apply

1. Use **Apply** on `/admin/settings/backup` with confirmation `APPLY BACKUP`.
2. **Restart** the backend/container so pending files in `_restore_pending` are applied.
3. Verify in the web UI (SOPs, acknowledgments, audit as expected).

### From a manual directory copy

Restoring from a backup consists of replacing the data directory.

- Stop the sopandgo container
- Replace the data directory with a backup copy
- Start the container

After restore:

- all SOPs, versions, acknowledgments, and audit events (as of backup time) are restored

### Verifying a restore

After restoring:

- open the web interface
- verify that SOPs and versions are visible
- check that acknowledgments are present
- confirm that recent changes match expectations

If integrity checks are enabled, sopandgo may warn if file hashes do not match
recorded metadata.

---

## Migration and portability

Because SOP content is stored as plain files:

- SOPs remain readable even outside sopandgo
- backups can be inspected without special tools
- long-term archiving is straightforward

**Schema compatibility:** Restoring an **older** backup into a **newer** app version
is supported (migrations upgrade the database on startup). Restoring a backup
whose schema is **newer** than the app supports is rejected by validate/apply.

---

## What is not backed up automatically

Unless **S3 automatic backups** are enabled, sopandgo does not upload archives by
itself. In all cases, operators must still preserve:

- configuration **outside** the data directory (e.g. `docker-compose.yml`, `ORIGIN`, `SMTP_SECRET_ENCRYPTION_KEY`)
- container images
- host system settings

Operators are responsible for preserving these as needed.
