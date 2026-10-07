# Deployment

This document describes how to deploy sopandgo in a typical self-hosted environment. It is intended for administrators and technical staff responsible for running the system in a lab, university, or small organization.

Trying the product locally with sample data first? See [Evaluating sopandgo](./evaluating.md). Version tags and GitHub Releases: [Releasing](./releasing.md).

## Deployment model

sopandgo is designed to run as a **single self-hosted service**.

Key properties:
- **App container**: One image (**`sopandgo`**) runs Caddy, the Go API, and the SvelteKit Node server together.
- **PDF (optional)**: Compose also runs **Gotenberg** by default when PDF export is enabled; see root `docker-compose.yml` and `.env.example`.
- **Persistent Storage**: One data volume for all database and file assets.
- **Configuration**: Sane defaults in compose; override with a root **`.env`** (from **`.env.example`**) for `ORIGIN`, mail encryption key, ports, and PDF-related variables.

This model is intentional and optimized for simplicity, portability, and reliability.

## Data directory

All persistent data lives under **`DATA_DIR`** inside the process (default **`/app/data`** in Docker). On the default compose setup, that is **`./backend/data` on the host** bind-mounted to **`/app/data`** in the **`sopandgo`** container.

The directory contains:

- `app.db`: The SQLite database.
- `sops/`: SOP content — versioned Markdown and per-SOP assets (images, attachments) on disk.

**Important:** This directory **must** be on persistent storage so data survives image updates and container recreation.

## Basic deployment (recommended)

### Using docker compose
A **`docker-compose.yml`** is in the repository root ( **`sopandgo`** app + **`gotenberg`** for PDF by default). Copy **`.env.example`** to **`.env`** and set variables there; Compose substitutes `${VAR}` from `.env` automatically.

Start the stack. The compose file uses the published image `ghcr.io/sopandgo/sopandgo:1`. You do not choose a version number.

```sh
docker compose pull
docker compose up -d
```

Initial Credentials:

- User: `admin`
- Password: `admin`
- **Forced change:** the bootstrap admin must set a new password on first login before other routes unlock.

Demo seed users appear on Docker first boot only when `SEED_DEMO_DATA` is left on (default `true`). They use weak passwords such as `12345` for exploration only. Set `SEED_DEMO_DATA=false` before the first start to skip them. Changing the flag later does not remove data that was already seeded.

## Advanced Configuration

### Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `DATA_DIR` | `/app/data` | The internal path for SQLite and assets. |
| `SEED_DEMO_DATA` | `true` | When not `false`, the first start with no `app.db` inserts demo users and a sample SOP. Set `false` before first boot for a real deployment. Has no effect once `app.db` exists. |
| `ORIGIN` | `http://localhost:8087` | **CRITICAL.** The full public URL of your instance. If this does not match the browser URL, login/invites will fail. |
| `HOST_WEB_PORT` | `8087` | Host port mapped to Caddy `:80` inside the `sopandgo` container (compose). |
| `BACKEND_URL` | `http://127.0.0.1:8080` | Internal URL the SvelteKit Node server uses to reach the Go API (loopback inside the container). |
| `HOST_HEADER` | `x-forwarded-host` | Forwarded host header name for SvelteKit behind Caddy. |
| `PROTOCOL_HEADER` | `x-forwarded-proto` | Forwarded protocol header name for SvelteKit behind Caddy. |
| `BODY_SIZE_LIMIT` | `52428800` | Max request size (bytes). Default is ~50MB. |
| `AUDIT_STRICT_TYPES` | `false` | If `true`, audit writes reject unknown `event_type`/`entity_type` values (recommended once all extensions use canonical types). |
| `AUDIT_BUSY_TIMEOUT` | `5s` | Maximum SQLite lock wait for standalone audit writes, as a Go duration. A timeout is reported in the application log; increase this only if legitimate write transactions regularly exceed five seconds. |
| `SMTP_SECRET_ENCRYPTION_KEY` | - | **Required to save outbound secrets in the UI** (SMTP password, Resend API key, Slack webhook URL, Gotify token, optional webhook bearer) **and to send mail or integration notifications that need those secrets.** A 32-byte AES-256 key, provided as **base64** or **hex** (64 hex chars, optional `0x` prefix). Used only to encrypt secrets stored in SQLite—not for signing JWTs. Generate: `openssl rand -base64 32`. If unset, the server starts, but the admin UI cannot persist those credentials until this is set and the process is restarted. |
| `APP_VERSION` | baked into the image | Label on the public footer, in the signed-in sidebar, and in admin **export** `manifest.json`. Release images set this at build time (for example `1.0.1`). Labs do not set it. Contributor source builds and `go run` use `dev`. Local `npm run dev` ignores this and shows `git describe` instead. The root `package.json` version is not used. |
| `PDF_EXPORT_ENABLED` | `true` | When `true`, PDF artifact generation/download is enabled (requires a renderer when not `none`). |
| `PDF_RENDERER` | `gotenberg` | `gotenberg` or `none`. |
| `GOTENBERG_URL` | `http://gotenberg:3000` | Base URL of the Gotenberg service (default compose service name: `gotenberg`). |
| `PDF_GENERATOR_VERSION` | `1` | Integer version for PDF pipeline/backfill; bump when output format changes. |
| `BACKUP_S3_ENABLED` | `false` | When `true`, schedule automatic full-data `.zip` uploads to S3 (same archive as Admin → Export). See `docs/ops/backup-and-restore.md`. |
| `BACKUP_S3_BUCKET` | - | Target bucket (required when S3 backups are enabled). |
| `BACKUP_S3_REGION` | `us-east-1` | AWS region (or compatible). |
| `BACKUP_S3_PREFIX` | - | Optional key prefix (e.g. `myorg/prod`). |
| `BACKUP_S3_INTERVAL` | `24h` | How often to upload (Go duration). |
| `BACKUP_S3_RETENTION_MAX` | `14` | Max number of remote backups to keep (0 = unlimited by count). |
| `BACKUP_S3_RETENTION_DAYS` | `30` | Delete remote backups older than this many days (0 = unlimited by age). |
| `BACKUP_S3_ENDPOINT` | - | Optional custom endpoint (MinIO / S3-compatible). |
| `BACKUP_S3_USE_PATH_STYLE` | `false` | Path-style addressing for compatible endpoints. |
| `BACKUP_S3_ACCESS_KEY_ID` / `BACKUP_S3_SECRET_ACCESS_KEY` | - | Optional static credentials; if both empty, the AWS SDK default chain is used (e.g. IAM role). |

The Go backend does **not** read a `.env` file on its own. With **Docker Compose**, put values in a root **`.env`** (substituted into `docker-compose.yml`) and/or under `environment:` in the compose file. For local **`go run`**, export variables in your shell or IDE run configuration.

### Email (SMTP, Resend, or manual links)

Outgoing mail (user invites, password resets) is **not** configured via `SMTP_HOST` / `SMTP_USER` style environment variables. Credentials live in SQLite and are encrypted with `SMTP_SECRET_ENCRYPTION_KEY`.

**Mail delivery mode** (admin UI, **Mail delivery mode** on `/admin/settings/email`):

- `smtp` (default): invite/reset links are sent by email when a transport is configured and working.
- `manual_links`: invite/reset links are returned to admins in the UI/API for manual sharing over a trusted channel.

**Outbound transport** (when not in `manual_links`): choose **SMTP** (your own server) or **Resend** ([resend.com](https://resend.com) API). Only the selected transport is used for sends and for **Send test email**.

1. Set `SMTP_SECRET_ENCRYPTION_KEY` as above (required to save any stored secret and to send).
2. Sign in as **admin**, open **Settings → Email** (`/admin/settings/email`).
3. For **SMTP**: enter host, port, username, password, and from-address, then save.
4. For **Resend**: create an API key in the Resend dashboard, set **Outbound transport** to Resend, enter from-address and API key, then save. Use a verified domain (or Resend’s test sender for trials).
5. Use **Send test email** to verify connectivity.

If you use `manual_links`, SMTP and Resend fields are optional until you switch back to email delivery.

**What mail does in 1.0:** invites, password resets, test email, and **optional notices when a version is published** (to other active users who can reader-sign). Publish notices are skipped in `manual_links` mode; a failed send does **not** unpublish the version. Pending-acknowledgment reminder digests and emails for RC/reject/archive are **not** included — use the home dashboard and training coverage views instead.

### Outbound integrations (Slack, Gotify, webhook)

Channel credentials are **not** set via env vars (no `SLACK_WEBHOOK_URL` / `GOTIFY_*` feature flags). Configure them under **Settings → Integrations** (`/admin/settings/integrations`), encrypted with the same `SMTP_SECRET_ENCRYPTION_KEY`.

| Channel | What you configure | Delivery |
| --- | --- | --- |
| Slack | Incoming Webhook URL (HTTPS), enable toggle, event checkboxes | POST `{"text":"..."}` |
| Gotify | Server base URL + application token, enable, events | POST `{url}/message?token=...` |
| Generic webhook | Destination URL, optional bearer token (enter `-` to clear), enable, events | POST JSON envelope (`event`, `occurred_at`, `title`, `message`, `url`, `sop_id`, …) |

**Events:** `sop_published`, `sop_rc`, `sop_rejected`, `backup_s3_failed`, `integrity_check_failed`. Use **Send test** per channel to verify. Delivery failures are audited and never roll back SOP publish/promote/reject.

If you **rotate** `SMTP_SECRET_ENCRYPTION_KEY`, existing ciphertext becomes undecryptable; sign in as admin and **re-enter the SMTP password and/or Resend API key**, plus any Slack/Gotify/webhook secrets, then save.

### Audit Taxonomy Hardening

Audit taxonomy is now normalized to lowercase (`event_type`, `entity_type`) at write time and via migration for existing rows.

- Keep `AUDIT_STRICT_TYPES=false` during transition if custom integrations still emit ad-hoc values.
- Set `AUDIT_STRICT_TYPES=true` to enforce known canonical values and prevent taxonomy drift.

### Network Access & Security

**sopandgo** includes an internal Caddy proxy to handle request routing.

* **Direct API Access:** You can directly access the Go backend via `/sopandgo/api/*` (Caddy forwards to Go on **:8080** inside the container).
* **Frontend:** All other traffic is handled by the SvelteKit Node.js adapter on **:3000** inside the container (behind Caddy on **:80**).

**Security Recommendations:**

The application does not enforce network-level restrictions. Operators should:

- **Restrict Access**: Use a firewall or institutional network access control.

- **VPN Deployment**: Deploy behind a VPN (like Tailscale or WireGuard) rather than opening ports to the public internet.

- **HTTPS/TLS:** If exposing to a network, terminate TLS at an external reverse proxy and forward **HTTP** to the **published app port** (compose default **`HOST_WEB_PORT` 8087** → Caddy **:80** inside the **`sopandgo`** container). Do **not** point the outer proxy at the Go process port **8080** unless you intentionally bypass Caddy (unsupported for normal deployments).

## Maintenance

### Backup

**Recommended:** Sign in as **admin** and use **Settings → Backup** (`/admin/settings/backup`) to **export** a `.zip` (consistent DB snapshot plus `sops/` and `manifest.json`). You can **validate** archives and **stage** a restore; staged restores apply on the **next backend restart**. See `docs/ops/backup-and-restore.md`.

**Manual:** Stop the container and copy the entire mounted data directory, or at minimum `app.db` and the `sops/` tree together (SQLite may use `app.db-wal` / `app.db-shm` while running—stopping the container avoids inconsistent copies).

During backup export, mutating API requests may return `503` briefly while a maintenance lock is active; reads generally continue.

### Upgrading

From **1.0.0** onward, the image tag in `docker-compose.yml` is `ghcr.io/sopandgo/sopandgo:1`. That name moves to each new 1.x release. A normal update does not edit the tag.

1. Export a backup (admin UI) or stop the container and copy the data directory.
2. `docker compose pull` and `docker compose up -d`.
3. Append-only migrations in `backend/internal/storage/migrations.go` apply automatically on startup.
4. Verify login and a published SOP. After sign-in, the sidebar shows the exact release, such as `v1.0.1`. Public pages still show it in the footer.

### Rollback and a future major version

These are not part of a normal install or update.

- **Roll back** to an exact release by changing the image tag in `docker-compose.yml` from `1` to that release, for example `ghcr.io/sopandgo/sopandgo:1.0.1`, then pull and start. If the database was already migrated by a newer release, restore the backup instead. Do not point an older binary at a database that has already been migrated.
- **Move to 2.x** only when a release note says to. Change the image tag from `1` to `2`. Until then, leave it at `1`. The name `latest` can move to 2.0.0; the lab tag `1` does not.
