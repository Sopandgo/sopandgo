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
- **`SMTP_SECRET_ENCRYPTION_KEY`** — 32-byte AES key as base64 or hex (`openssl rand -base64 32`)
- Leave **`SEED_DEMO_DATA=true`** (the default) so the first boot inserts demo users and a sample SOP

3. Start the stack. This pulls the published image. You do not set a version, check out a tag, or build from source.

```bash
docker compose pull
docker compose up -d
```

4. Open **`ORIGIN`** (default [http://localhost:8087](http://localhost:8087)).

## Who to sign in as

| Account | Credentials | Role | Use for |
| --- | --- | --- | --- |
| Bootstrap admin | `admin` / `admin` | Admin | Settings, users, backup, audit. **Must change password on first login** before the rest of the app unlocks. |
| Demo Lab Manager | `manager@demo.local` / `12345` | Editor | Drafts and authoring |
| Demo QA Officer | `qa@demo.local` / `12345` | Approver | Review / publish path |
| Demo Research Assistant | `researcher@demo.local` / `12345` | Viewer | Reading and acknowledgments |

Demo accounts are for local exploration only. For a real lab, set **`SEED_DEMO_DATA=false`** in `.env` **before** the first boot so demo users and the sample SOP are not inserted. Changing the flag later does not remove data already seeded.

## Ten-minute tour

1. Sign in as **`researcher@demo.local`** (or change the admin password, then explore as admin).
2. Open the **home dashboard** — pending signatures, favorites, and recently published versions.
3. Browse the **library** (`/sops`) and open the seeded sample SOP.
4. Open a published version: read the Markdown, check acknowledgments / training coverage if your role allows it.
5. Sign in as **`manager@demo.local`** and create or edit a **draft** (Word import is available in the editor when you want to try it).
6. Sign in as **`qa@demo.local`** (or admin) and walk a version through the lifecycle toward **published** if you want to see RC → publish and reader signs.
7. As **admin**, skim **Settings** (language, mail mode, optional Slack / Gotify / webhook integrations, and **Backup** — export a `.zip` once so you know the path works) and **Audit**.

You do not need real SMTP for a first look: use **manual links** mail mode in Settings if you invite another user.

Before sign-in, the top bar has a language menu and a light/dark toggle. Supported locales include English, German, French, Spanish, Portuguese, Chinese, Italian, Dutch, Polish, Japanese, Korean, Turkish, and Swedish. The appearance choice stays in the browser until sign-in. After sign-in, change your language and appearance (light, dark, or system) on **Profile → Account settings**. Admins set the organization language on **Settings**; that language is used for shared notifications and PDF chrome. The seeded sample SOP stays in the English Markdown it was written in.

## After the trial

- Tear down with `docker compose down` (add `-v` only if you also want to remove named volumes; the default bind mount `./backend/data` remains on disk until you delete it).
- For a real deployment, start from a clean data directory with `SEED_DEMO_DATA=false`, set a strong admin password, and follow [deployment.md](./deployment.md) and the README first-week checklist.

## Related docs

- [Deployment](./deployment.md) — environment variables and day-two ops
- [Security model](../concepts/security-model.md) — auth and invitation assumptions
- [Versioning and signatures](../concepts/versioning-and-signatures.md) — draft → RC → published
- [Home dashboard](../concepts/home-dashboard.md) — what the landing page after login shows
