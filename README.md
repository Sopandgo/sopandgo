# sopandgo
> [!NOTE]
> **Stability:** From **1.0.0** onward, upgrades are intended to be forward-compatible: take a backup, pull the image, restart, and let append-only database migrations run. The compose file stays on version **1**. See [Upgrading](#upgrading) below.

A small, self-hosted SOP management system for labs and small research organizations.

---

## What this is

sopandgo is a lightweight tool for managing **standard operating procedures (SOPs)** as
structured, versioned documents with explicit acknowledgment and auditability.

It is designed for environments where SOPs matter, but full QMS software is unnecessary
or impractical — such as academic labs, core facilities, and early-stage research teams.

Core ideas:
- SOPs are **structured documents**, not loose files.
- Version **content** is **immutable**; publishing uses an explicit **draft → RC → published** lifecycle.
- Acknowledgments are tied to **exact versions**.
- Users can **bookmark SOPs as favorites** (per-user, private to each account; favorites apply to the SOP container, not a specific version).
- All data stays **self-hosted and portable**.

---

## What this is not

- Not a Quality Management System (QMS)
- Not compliance or validation software
- Not a document editor replacement
- Not cloud-hosted

sopandgo does **not** claim to make an organization compliant with GMP, GLP, ISO, or any
other regulatory framework. It is a tooling aid, not a regulatory solution.

---

## High-level architecture

The system follows a **layered architecture** pattern to keep transport, services, and storage separable for security and portability.

- **Containerization:** Root **`docker-compose.yml`** (use with a **`.env`** file; start from **`.env.example`**):
  - **`sopandgo` service (one image):** Caddy (reverse proxy), **Backend:** Go 1.25 (service layer, SQLite, PASETO auth, mail hooks), **Frontend:** Svelte 5 / SvelteKit (Node adapter).
  - **`gotenberg` service:** PDF rendering sidecar when PDF export is enabled (see compose file and env vars below).
- **Storage:**
  - **SQLite:** Stores users, sessions, metadata, audit logs, and encrypted secrets for mail and outbound integrations (ciphertext only; the AES key stays in the environment).
  - **Filesystem:** Stores raw Markdown content and binary assets.

If the application is removed, SOP content remains readable on disk as standard files.

---

## SOP model (brief)

- Each SOP consists of **versions** with append-only content
- Each version:
  - has structured content (Markdown) that is never rewritten after create
  - may reference local assets (images, diagrams)
  - has a content hash recorded in the database; any user can re-check a version or asset file against it, and admins can scan everything at **Admin → Integrity** (`/admin/integrity`)
  - moves through a lifecycle: **draft → RC → published** (or rejected); a later publish marks the prior published version **superseded**
- Users acknowledge specific SOP versions (`author` / `approver` / `reader`)
- Content files are never modified or deleted; lifecycle state history is recorded separately

---

## Security & access

The system implements a **Stateless/Stateful Hybrid** security model:

- **Unknowable Passwords:** Admins cannot set passwords. Users set their own credentials via email invites.
- **Authentication:** PASETO (Stateless) for fast API access + Refresh Tokens (Stateful) for session management.
- **Revocation:** Deactivating a user or changing their role applies on the next request. Revoking a refresh session stops new tokens immediately; an access token already issued lasts at most 5 minutes.
- **RBAC:** Role-Based Access Control limits actions (e.g., only "Editors" can create drafts).
- **Self-Contained:** No external identity providers required.

The security model is documented in detail under `docs/concepts/security-model.md`.

---

## Features (1.0)

Self-hosted SOP management with invite-only auth, RBAC, immutable versioned content (draft → RC → published), acknowledgments, home dashboard, tags and favorites, audit trail, mail (SMTP/Resend or manual links), optional publish notices, Slack/Gotify/HTTP webhooks, backup/restore, PDF export, and Word import. The interface, emails, notifications, and PDF chrome ship in English, German, French, Spanish, Portuguese, Chinese, Italian, Dutch, Polish, Japanese, Korean, Turkish, and Swedish. SOP titles and Markdown stay in the language the author wrote. Adding another interface language is a catalog change; see [Architecture](docs/dev/architecture.md#languages).

Pending-acknowledgment reminder digests and lifecycle emails for RC/reject/archive are **out of scope for 1.0** — the dashboard and training coverage are the nudge surfaces; publish notices are the email surface.

Release notes: [CHANGELOG.md](CHANGELOG.md). Local evaluation: [docs/ops/evaluating.md](docs/ops/evaluating.md).

---

## Deployment

**Recommended:** run with **`docker compose`** using the bundled **`docker-compose.yml`** plus a **`.env`** in the repo root (copy **`.env.example`** → **`.env`** and edit; do not commit `.env`). Compose loads `.env` automatically for `${VAR}` substitution in the compose file.

- **`sopandgo`:** one container image (Caddy + Go API + SvelteKit Node server).
- **`gotenberg`:** second container for PDF generation when enabled (default in compose). Disable PDF or adjust `PDF_EXPORT_ENABLED` / `PDF_RENDERER` if you do not want it.
- **Networking:** one **host** port mapped to the app (default **8087** → container port 80; override with **`HOST_WEB_PORT`** in `.env`).
- **Data:** one bind-mounted directory for SQLite and SOP files (default **`./backend/data`** → `/app/data` in the container).

### Quick Start

Trying the product for the first time? See **[Evaluating sopandgo](docs/ops/evaluating.md)** for demo accounts and a short tour. For a real lab, follow the steps below with **`SEED_DEMO_DATA=false`** before first boot.

1. **Clone the repo**
```bash
  git clone https://github.com/Sopandgo/sopandgo.git
  cd sopandgo
```
2. **Configure environment (critical)** Copy **`.env.example`** to **`.env`** and set at least the values you need for your deployment. The repo already includes **`docker-compose.yml`**; you normally do **not** need to author compose from scratch.

- **ORIGIN:** Must match the URL you use in the browser (default: `http://localhost:8087`). If it does not match, login fails with **403 Forbidden**.
- **SMTP_SECRET_ENCRYPTION_KEY:** A **32-byte** AES key as **base64** or **hex** (generate: `openssl rand -base64 32`). This encrypts secrets stored in SQLite (SMTP password, Resend API key, Slack webhook URL, Gotify token, optional webhook bearer); it is **not** your mail provider password. Without a valid key, the server starts, but you cannot save those secrets or send mail/integrations that need them.
- **SEED_DEMO_DATA:** `true` (default) inserts demo users and a sample SOP on the first Docker start, when `app.db` does not exist yet. Set `false` for a real deployment. Changing it later does not remove data that was already seeded.
- **Mail mode:** In **Settings → Email** (`/admin/settings/email`), choose either `smtp` (default; sends invites/resets by email) or `manual_links` (admin copies one-time links and shares them manually).
- **SMTP provider details** (host, port, user, password, from-address) are configured on that Email page when using `smtp`, then verified with **Send test email**.
- **Integrations:** On **Settings → Integrations** (`/admin/settings/integrations`), optionally enable Slack Incoming Webhooks, Gotify, and/or a generic HTTP webhook for lifecycle and ops alerts. There are no `SLACK_ENABLED`-style env flags — configure destinations in the UI.
- **PDF export (Gotenberg):**
  - `PDF_EXPORT_ENABLED=true|false` toggles PDF artifact generation and download.
  - `PDF_RENDERER=gotenberg|none` selects renderer mode.
  - `GOTENBERG_URL` points to the Gotenberg service URL (default in Docker: `http://gotenberg:3000`).
  - `PDF_GENERATOR_VERSION` controls append-only artifact generation/backfill versioning.
- **Backups:** Admins can use **Settings → Backup** (`/admin/settings/backup`) to export a `.zip`, validate archives, and stage a restore (restart required). Optional automatic uploads to S3 use `BACKUP_S3_*` in `.env` (see `docs/ops/backup-and-restore.md`).

See `docs/ops/deployment.md` for the full environment variable reference. How versions ship: `docs/ops/releasing.md`.

3. **Launch**
```bash
  docker compose pull
  docker compose up -d
```
The compose file uses the published image `ghcr.io/sopandgo/sopandgo:1`. You do not set a version. (`docker-compose` with a hyphen also works if that is what your system provides.)

4. **Access the app**

Open the URL matching **`ORIGIN`** (default: http://localhost:8087; use your **`HOST_WEB_PORT`** if you changed it).

**Default Admin:** `admin` / `admin`

> [!IMPORTANT]
> The bootstrap admin **must change this password on first login** before the rest of the app unlocks. The UI redirects you to the profile password form until you do.

**Demo users** (seeded on first Docker start when `SEED_DEMO_DATA` is not `false`): emails like `manager@demo.local` with password `12345`. Demo accounts are for exploration only — set `SEED_DEMO_DATA=false` before first boot for a real lab deployment.

### First-week checklist

1. Set **`ORIGIN`** and **`SMTP_SECRET_ENCRYPTION_KEY`** in `.env`. For a real deployment, set **`SEED_DEMO_DATA=false`** before the first boot so demo users and the sample SOP are not inserted. Restart if you changed `ORIGIN` or the encryption key after first boot. The seed flag has no effect once `app.db` exists.
2. Sign in as `admin` / `admin` and **set a strong password** (required).
3. Configure mail (`smtp` + transport, or `manual_links`) and send a **test email** if using SMTP/Resend.
4. Create a real user invite (or keep demo data only for a trial).
5. **Backup:** Admin → **Backup** → export a `.zip`, then optionally **validate** it.
6. Practice restore once on a non-production copy: stage apply → restart container → confirm SOPs and acknowledgments. See `docs/ops/backup-and-restore.md`.

### Upgrading

1. Export a backup from **Backup** (or copy `backend/data` while the container is stopped).
2. Pull and start again:

```bash
docker compose pull
docker compose up -d
```

3. The backend applies **append-only** schema migrations automatically.
4. Confirm the app boots and spot-check a published SOP and a recent backup export. The footer shows the exact release, such as `v1.0.1`.

You do not edit a version number for a normal update. The image name `1` moves forward for compatible releases and does not jump to 2.0.0. To roll back, or to move to a future major version, see [docs/ops/deployment.md](docs/ops/deployment.md).

Do not run a newer database against an older application binary; restore from backup if you need to roll back the app version.

---

## Data Persistence & Backups

All system data is stored in the `backend/data` folder (inside the container: `DATA_DIR`, default `/app/data`), which is mounted as a persistent volume in Docker.

- **SQLite Database:** `app.db` holds metadata, audit logs, users, per-user SOP favorites (`sop_favorites`), and encrypted SMTP / integration settings.
- **SOP content:** `sops/` holds versioned Markdown and per-SOP assets.

**Recommended (admin UI):** Sign in as **admin** → **Settings → Backup** (`/admin/settings/backup`). You can **export** a `.zip` (consistent DB snapshot plus `sops/` and `manifest.json`), **validate** an archive, and **stage apply** of a restore. Staged restores run on the **next backend/container restart**; see `docs/ops/backup-and-restore.md`.

**Alternative (manual):** Stop the container and copy the whole data directory, or archive `backend/data` on the host. SOP Markdown on disk stays readable without the app.

---

## Development

If you wish to run the services outside of Docker:
- **Backend:** `cd backend && go run cmd/sopandgo/main.go`  
  The Go binary does **not** read a `.env` file; export `SMTP_SECRET_ENCRYPTION_KEY` (and other vars) in your shell or IDE if you need mail or to match production.
- **Frontend:** `cd frontend && npm run dev`
- **Seeder:** `cd backend && go run cmd/seed-demo-data/main.go` (Only run on an empty `data` folder). Honors `SEED_DEMO_DATA=false`.

### Tests
- **Backend:** `cd backend && go test ./...`
- **Frontend:** `cd frontend && npm test`

See [CONTRIBUTING.md](CONTRIBUTING.md) for pull-request expectations.

---

## Disclaimer

This software is provided as-is, without warranty of any kind.

It is not validated for regulated environments and does not replace institutional,
legal, or regulatory responsibilities. Users are responsible for determining whether
its use is appropriate for their context.

---

## License

MIT License  
© 2025-2026 Leopold Dürrauer

This is an independent open-source project and is not affiliated with any employer or
institution.

### Third-Party Assets
The demo data bundled with this application includes media files used under 
different licenses:
* Sample images are sourced from Wikimedia Commons and are licensed under 
  Creative Commons (CC BY 4.0 / CC BY-SA 3.0). 
* See `backend/demo/sops/87de1162-589b-4c91-8c37-8fae346a1ccc/assets/ATTRIBUTIONS.md` for full credits and license links used in the sopandgo backend.
* The Gopher image is based on the Go Gopher created by Renee French. Licensed under Creative Commons Attribution 4.0 International License. https://creativecommons.org/licenses/by/4.0/
