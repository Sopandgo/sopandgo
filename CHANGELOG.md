# Changelog

All notable releases of sopandgo are documented here. The project follows semantic versioning for tagged releases from **1.0.0** onward.

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
