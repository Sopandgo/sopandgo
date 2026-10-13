# Changelog

All notable releases of SOPandGO are documented here. The project follows semantic versioning for tagged releases from **1.0.0** onward.

## 1.2.2

Publishes the 1.2.1 sign-in fix. The v1.2.1 image was not published: compiling translations for the arm64 image timed out under emulation.

- The frontend is compiled on the machine that runs the image build. That output is copied into the arm64 image.

### Upgrading from 1.2.0

- No migrations. Export a backup, then `docker compose pull` and `docker compose up -d`. There is no 1.2.1 image.

## 1.2.1

Signing in on a local Docker install works when SOPandGO is published on a port other than 80.

No container image was published for this tag. Use 1.2.2.

- Caddy keeps the browser host and port when no outer proxy sets `X-Forwarded-Host`. A login at `http://localhost:8087` is no longer rejected as a cross-site form. Coolify and other proxies that send `X-Forwarded-Host` are unchanged.

### Upgrading from 1.2.0

- No migrations. Export a backup, then `docker compose pull` and `docker compose up -d`.

## 1.2.0

Labs can require a second sign-in factor, archive SOPs that are no longer used, and brand SOPandGO with their name, logo, and colour.

- Users can add an authenticator app. Under Settings → General → Security, an admin chooses the organization's required second factor: none (the default), an email code or an app, or an app only. Authenticator secrets are sealed with `SECRET_ENCRYPTION_KEY`. Admins receive recovery codes. A locked-out admin is recovered on the host with `./sopandgo recover-admin`, which has no web page.
- An approver or admin can archive an SOP that is no longer used. Archiving deletes nothing. The SOP stays readable and becomes read-only, and the SOP list shows it under Archived. Unarchiving restores it as it was.
- Admins set the organization name, logo, and colour under Settings → Branding. Empty settings keep the SOPandGO defaults. The logo is stored in the database, so backups carry it. The sign-in page shows the name and logo.
- The home dashboard lists each person's own pending signatures, recent publishes, and favorites. It no longer lists who else in the lab still needs to sign.
- Alerts use a status tint and border so they stay distinct from the page behind them.
- The Markdown sanitizer is updated (`isomorphic-dompurify` 4.5.0).

### Upgrading from 1.1.x

- Migrations run on startup. Take a backup before pulling, as with any upgrade.
- Sign-in stays password-only until an admin sets a two-factor policy. Raising the policy needs `SECRET_ENCRYPTION_KEY` and an admin who already uses an authenticator app. The email level also needs a switched-on email transport.
- Reset a locked-out admin from a shell in the container with `./sopandgo recover-admin`. See [Deployment](docs/ops/deployment.md#locked-out-admin-break-glass).

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

From 1.0.0 onward, upgrades are intended to be forward-compatible: take a backup, pull the new image, restart, and let append-only database migrations run. Prefer tagged releases. See [Upgrading](docs/ops/deployment.md#upgrading).
