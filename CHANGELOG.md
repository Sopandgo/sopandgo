# Changelog

All notable releases of sopandgo are documented here. The project follows semantic versioning for tagged releases from **1.0.0** onward.

## 1.1.1

Publishes the 1.1.0 changes. The v1.1.0 image was not published: the Caddy apt repository returned HTTP 402 during the build.

- Install Caddy from the official `caddy:2.11.7` image instead of the Cloudsmith apt repository.
- New signed-in shell: a sidebar, breadcrumbs, and an avatar menu for profile, theme, and sign out. `/` opens the dashboard or the login page.
- Users can set a profile picture. It is included in backups, and the avatar falls back to initials.
- Admin settings are split into Email, Integrations, and Backup. Email transports can be turned off; manual invite links remain the fallback.
- Scheduled S3 backups, a connection test, and run-now live under Settings → Backup. The secret is sealed with `SECRET_ENCRYPTION_KEY`. A staged restore keeps the running instance's S3 settings.
- Admin → Integrity scans every version, every asset, and the audit chain.
- A rejection stores the reason. The version page shows who rejected the version, when, and why.
- Users change their own password with the current password, and can sign out their other sessions.
- Czech and Slovak join the interface languages.
- A second reader acknowledgment of the same version returns 409 instead of 500.
- With `SEED_DEMO_DATA`, a first boot seeds six sample SOPs covering tags, favorites, a draft, a release candidate, and signature states.
- Frontend updates clear the undici and DOMPurify security advisories.

### Upgrading from 1.0.x

- `BACKUP_S3_*` environment variables are no longer read. After the upgrade, set scheduled S3 backups again under Admin → Settings → Backup.
- Only `SECRET_ENCRYPTION_KEY` is read. The `SMTP_SECRET_ENCRYPTION_KEY` name is no longer accepted.
- SvelteKit no longer reads `ORIGIN`. The Go API still uses it for invite links and the public URL. CSRF uses `HOST_HEADER` and `PROTOCOL_HEADER`, which the bundled compose file already sets.
- Migrations run on startup. Take a backup before pulling, as with any upgrade.

## 1.1.0

No container image was published for this tag. Use 1.1.1.

## 1.0.2

Standalone audit writes no longer race under concurrent SQLite writers.

- Serialize audit writes that run without a caller transaction, and take SQLite's write lock before reading the chain tip, so concurrent writers cannot fork the hash chain.
- Bound lock waits with `AUDIT_BUSY_TIMEOUT` (default `5s`); timed-out or failed audit writes are reported in the application log.
- Document the behavior and env var for operators and developers.

## 1.0.1

The interface, emails, notifications, and PDF chrome now ship in English, German, French, Spanish, Portuguese, Chinese, Italian, Dutch, Polish, Japanese, Korean, Turkish, and Swedish. SOP titles and Markdown stay in the language the author wrote.

- Each user can set a preferred language. Admins set the organization language used for shared notifications and PDF chrome.
- Profile includes a light, dark, or system theme preference.
- The change-password and settings screens use a layout that fits smaller windows.

## 1.0.0

First stable release for self-hosted labs and small research teams.

### Included

- Authentication (PASETO access tokens + refresh sessions), invite-only users, and first-login password change for the bootstrap admin
- Role-based access control for admin, editor, approver, and viewer workflows
- SOP containers with immutable version content, draft → RC → published (or rejected) lifecycle, and superseded history
- Acknowledgments tied to exact versions; training coverage for who still needs to reader-sign
- Home dashboard: pending signatures, favorites, recently published feed
- Tags, favorites, change summaries, and line diffs between versions
- Asset uploads, Markdown editing, Word (`.docx`) import, optional PDF export via Gotenberg
- Audit dashboard with filtering; mail (SMTP / Resend or manual invite links); publish-notice emails
- Outbound integrations: Slack Incoming Webhooks, Gotify, and generic HTTP webhooks
- Backup export, validate, and staged restore; optional scheduled S3 backup uploads
- Docker Compose deployment with optional `SEED_DEMO_DATA` for local evaluation

### Explicitly not in 1.0

- Pending-acknowledgment reminder digests
- Lifecycle emails for RC / reject / archive (dashboard and training coverage remain the nudge surfaces)
- SSO / LDAP, multi-tenant SaaS, interactive chat bots, LMS / quizzes, or QMS / compliance claims

### Upgrading

From 1.0.0 onward, upgrades are intended to be forward-compatible: take a backup, pull the new image, restart, and let append-only database migrations run. Prefer tagged releases. See the README upgrading section and [docs/ops/deployment.md](docs/ops/deployment.md).
