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
- Profile pictures — processed JPEGs under `users/<userID>/` (`avatar-96.jpg`, `avatar-256.jpg`, `avatar-512.jpg`, `avatar-1024.jpg`)

The SMTP password and integration secrets are stored encrypted; recovery still requires the same
`SECRET_ENCRYPTION_KEY` (or re-entered secrets after key rotation) when
an email transport or outbound integrations are switched on.
With no email transport on, invites/resets use manual links and do not need SMTP delivery.

If the full data directory is backed up, the entire system state is preserved.

---

## Admin UI backup (recommended for operators)

Sign in as **admin** and open **Settings → Backup**: `/admin/settings/backup`. `/admin/backup` redirects there.

### Export

- Downloads a `.zip` containing:
  - `app.db` — consistent SQLite snapshot (via `VACUUM INTO` while a short maintenance lock is held)
  - `sops/` — SOP files on disk (if present)
  - `users/` — profile pictures on disk (if present)
  - `manifest.json` — format version, optional `APP_VERSION`, schema version, timestamp

Export is also available via the API: `GET /api/admin/backups/export` (admin auth required).

### Validate

- Upload a `.zip` to check structure and compatibility **without** changing live data.
- Rejects backups whose **database schema is newer** than the running app (older-or-equal schema is allowed; migrations run on startup after restore).

### Restore (staged)

- Destructive: stages replacement of live data from the uploaded archive.
- After you choose the archive, a confirmation dialog repeats the warning and requires typing **`RESTORE BACKUP`** exactly.
- Before staging, the server writes a **pre-apply** snapshot zip under:
  - `data/_backup_snapshots/` (filename prefix `pre-apply-`)
- Staged payload is written to:
  - `data/_restore_pending/`
- **The restore is not active until the next backend process start.** Restart the
  container or process (e.g. `docker compose restart`) so startup can apply
  `_restore_pending` and then run normal migrations.
- While a restore is pending (`_restore_pending` present), further validate/restore
  uploads are blocked until after restart clears it.
- Automatic S3 backup settings are **not** taken from the archive: the running
  instance keeps its own (see [Automatic S3 backups](#automatic-s3-backups-optional)).

Environment variable (optional):

- **`APP_VERSION`** — recorded in export `manifest.json` for traceability. Release images bake this in at build time. Labs do not set it. It is `dev` when the process is started without that value.

---

## Automatic S3 backups (optional)

When switched on, the backend uploads the **same** `.zip` as manual export on a
fixed interval. Configure it under **Settings → Backup → Automatic S3 backups**
(`/admin/settings/backup`). There are no environment variables for it: settings
are stored in SQLite (`backup_s3_settings`) and take effect when you save, without
a restart.

Objects are stored under:

`<key prefix>/automated/sopandgo-automated-YYYYMMDDThhmmss.zip`

(The key prefix is optional; leading and trailing slashes are trimmed.)

| Setting | Meaning |
|---------|---------|
| **On/off toggle** | In the card header, once a bucket is saved; until then the header says "Not configured". Save the settings first, then switch it on. |
| **Bucket** | Target bucket name (required while on). |
| **Region** | AWS region for the SDK (set it for AWS; for MinIO a placeholder such as `us-east-1` works). |
| **Key prefix** | Optional, e.g. `prod/myorg`. |
| **Endpoint** | Optional custom endpoint (MinIO, Cloudflare R2, Backblaze B2, LocalStack…). Leave empty for AWS S3. |
| **Path-style URLs** | Path-style addressing, needed by most MinIO setups. |
| **Access key ID / secret access key** | Optional static credentials. Leave **both** empty to use the AWS SDK default chain (container or instance role, `AWS_*` variables in the backend environment, shared config). The secret is encrypted with `SECRET_ENCRYPTION_KEY` and never returned; leave it blank to keep the stored one. A new access key ID needs its secret. |
| **Schedule** | Every hour, 6 hours, 12 hours, day (default) or week. |
| **Keep at most** | After each successful upload, delete older automated archives until at most this many **non-expired** objects remain (`0` = no count limit; default `14`). |
| **Delete after (days)** | Delete automated archives whose S3 `LastModified` is older than this (`0` = no age limit; default `30`). |

**Test connection** writes and deletes a small probe object
(`<key prefix>/automated/sopandgo-connection-test.txt`) with the **saved**
settings, so it checks credentials, bucket and write permission without uploading
an archive. **Back up now** (while on) uploads one archive immediately and moves the
next scheduled run one interval out. Changing only retention keeps the next run
time; changing the schedule or switching on starts the interval again.

**Retention semantics:** objects matching the automated naming pattern under the
prefix are listed. Any object **older than** the age limit (when set) is deleted.
Among those that remain, if the count limit is set and the count is still greater
than the limit, the **oldest** extras are deleted until the count is at most the
limit. The connection-test probe never matches the pattern.

**Restores keep this instance's S3 settings.** The settings live in `app.db`,
which an archive contains, but a staged restore keeps the running instance's
settings (including the encrypted secret) instead of the archive's: a restore
brings back data, not where backups go. It also means a restore onto a server
with a different `SECRET_ENCRYPTION_KEY` never leaves an S3 secret it cannot decrypt.

The admin **Backup** page and `GET /api/admin/backups/status` include a
`s3_scheduled` object (bucket, prefix, interval, retention settings, last
success, next run, last error). `GET /api/admin/settings/backup-s3` returns the
saved settings. **Secrets are never returned.**

Successful uploads and failures are written to the **audit log**
(`backup_s3_uploaded`, `backup_s3_failed`), and failures also go to the
`backup_s3_failed` integration event. Every settings change is audited as
`backup_s3_settings_updated` (without the secret).

**Security notes:** an admin can point backups at any bucket, which sends a full
copy of the data there on every run. That is no more than an admin can already do
with **Export**, but it is persistent, so check `backup_s3_settings_updated` in the
audit log. Use a dedicated IAM principal with least privilege on the
bucket/prefix (`s3:PutObject`, `s3:ListBucket`, `s3:DeleteObject`); prefer
**versioning** and (for stronger ransomware resistance) **S3 Object Lock** or a
separate backup account. Manual export/download is unchanged and does not require
S3 configuration.

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

### From an admin staged restore

1. Use **Restore backup** on `/admin/settings/backup` and confirm with `RESTORE BACKUP`.
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
whose schema is **newer** than the app supports is rejected by validate and restore.

---

## What is not backed up automatically

Unless **S3 automatic backups** are enabled, sopandgo does not upload archives by
itself. In all cases, operators must still preserve:

- configuration **outside** the data directory (e.g. `docker-compose.yml`, `ORIGIN`, `SECRET_ENCRYPTION_KEY`)
- container images
- host system settings

Operators are responsible for preserving these as needed.
