# Evaluating sopandgo

Use this guide if you want to try sopandgo on your own machine before deploying it for a lab. There is no hosted multi-tenant demo — evaluation means running the Docker stack locally with sample data.

## Prerequisites

- Docker and Docker Compose
- About 10 minutes

## Quick start

1. Clone the repository:

```bash
git clone https://github.com/Sopandgo/sopandgo.git
cd sopandgo
```

2. Copy environment defaults and set the encryption key:

```bash
cp .env.example .env
```

Edit `.env` and set at least:

- **`ORIGIN`** — must match the URL you open in the browser (default `http://localhost:8087`)
- **`SECRET_ENCRYPTION_KEY`** — 32-byte AES key as base64 or hex (`openssl rand -base64 32`)
- Leave **`SEED_DEMO_DATA=true`** (the default) so the first boot inserts demo users and sample SOPs

3. Start the stack. This pulls the published image. You do not set a version, check out a tag, or build from source.

```bash
docker compose pull
docker compose up -d
```

4. Open **`ORIGIN`** (default [http://localhost:8087](http://localhost:8087)). It opens on the sign-in form.

## Who to sign in as

| Account | Credentials | Role | Use for |
| --- | --- | --- | --- |
| Bootstrap admin | `admin` / `admin` | Admin | Settings, users, backup, audit. **Must change password on first login** (current password: `admin`) before the rest of the app unlocks. |
| Demo Lab Manager | `manager@demo.local` / `12345` | Editor | Drafts and authoring |
| Demo QA Officer | `qa@demo.local` / `12345` | Approver | Review / publish path |
| Demo Research Assistant | `researcher@demo.local` / `12345` | Viewer | Reading and acknowledgments |

Demo accounts are for local exploration only. For a real lab, set **`SEED_DEMO_DATA=false`** in `.env` **before** the first boot so demo users and the sample SOPs are not inserted. Changing the flag later does not remove data already seeded.

## What the demo data contains

Six short sample SOPs, tagged *Molecular Biology*, *Safety*, *Equipment*, and *Quality*, in different lifecycle states:

| SOP | State | For the researcher |
| --- | --- | --- |
| Agarose Gel Electrophoresis | v2 published | Signed, favorite |
| Chemical Spill Response | v2 published | Signed v1 only, so v2 needs a new signature; favorite |
| Pipette Performance Check | v1 published | Not signed yet |
| Lab Waste Segregation | v1 published | Signed |
| Autoclave Operation | v1 published, v2 waiting for approval (RC) | Signed v1 |
| Bacterial Glycerol Stock Preparation | v1 draft | Nothing to sign yet |

## Ten-minute tour

1. Sign in as **`researcher@demo.local`** (or change the admin password, then explore as admin).
2. Open the **home dashboard** — two SOPs wait for a signature, plus favorites and recently published versions.
3. Browse the **library** (`/sops`), filter by a tag, and open a seeded SOP. On *Chemical Spill Response*, compare v1 with v2.
4. Open a published version: read the Markdown, check acknowledgments / training coverage if your role allows it.
5. Sign in as **`manager@demo.local`** and continue the *Bacterial Glycerol Stock Preparation* **draft** (Word import is available in the editor when you want to try it).
6. Sign in as **`qa@demo.local`** (or admin) and approve the waiting *Autoclave Operation* release candidate. The researcher then has a new SOP version to sign.
7. As **admin**, skim **Settings** (language, email, optional Slack / Gotify / webhook integrations, and **Backup** — export a `.zip` once so you know the path works), **Audit**, and **Integrity** (run the system check once; it re-hashes every file and verifies the audit chain).

You do not need real SMTP for a first look: with no email transport switched on, inviting another user gives you a **manual link** to share.

Before sign-in, the top bar has a language menu and a light/dark toggle. Supported locales include English, German, French, Spanish, Portuguese, Chinese, Italian, Dutch, Polish, Japanese, Korean, Turkish, Swedish, Czech, and Slovak. The appearance choice stays in the browser until sign-in. After sign-in, the avatar menu at the top right switches appearance (light, dark, or system) and links to **Account settings** (`/profile/settings`), where you can also change your language. Admins set the organization language on **Settings**; that language is used for shared notifications and PDF chrome. The seeded sample SOPs stay in the English Markdown they were written in.

## After the trial

- Tear down with `docker compose down` (add `-v` only if you also want to remove named volumes; the default bind mount `./backend/data` remains on disk until you delete it).
- For a real deployment, start from a clean data directory with `SEED_DEMO_DATA=false`, set a strong admin password, and follow [deployment.md](./deployment.md) and the README first-week checklist.

## Related docs

- [Deployment](./deployment.md) — environment variables and day-two ops
- [Security model](../concepts/security-model.md) — auth and invitation assumptions
- [Versioning and signatures](../concepts/versioning-and-signatures.md) — draft → RC → published
- [Home dashboard](../concepts/home-dashboard.md) — what the landing page after login shows
