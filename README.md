# SOPandGO

Website: [sopandgo.com](https://sopandgo.com)

A self-hosted home for a lab's procedures: write them, publish a version, and have people sign the one they read.

---

## What this is

SOPandGO keeps each **standard operating procedure (SOP)** as a structured document with a clear history. It is for academic labs, core facilities, and early-stage research teams.

- Version content is immutable. Publishing follows **draft → release candidate → published**.
- Acknowledgments are tied to an exact version.
- Approvers can archive an SOP that is no longer used. It leaves the list and becomes read-only, and nothing is deleted.
- All data stays on the host.

## What a lab does with it

- Invite the people who need access. Each person sets their own password. Roles decide who drafts, who approves, and who reads and signs.
- Write in the editor, or start from a Word document. Attach images, tag procedures, and compare two versions.
- Send a draft through review, then publish. People can read it in the app or download a PDF.
- The home page shows each person what they still need to sign, what was just published, and the procedures they use often.
- Look back at who signed which version, and at the audit trail of publishes and admin actions.
- Export a backup from **Settings**, and restore it when you want to rehearse on another copy.

You can require a second sign-in factor, mail a notice when a version is published, and show your organization's name and logo on the sign-in page.

From **1.0.0** onward, upgrades are intended to be forward-compatible. See [Upgrading](docs/ops/deployment.md#upgrading).

## What this is not

- Not a Quality Management System (QMS)
- Not compliance or validation software
- Not a document editor replacement
- Not cloud-hosted

SOPandGO does not claim to make an organization compliant with GMP, GLP, ISO, or any other regulatory framework. It is a tooling aid for the people who run the lab.

## Run it

```bash
cp .env.example .env
```

Set `ORIGIN` to the URL you open in the browser, and set `SECRET_ENCRYPTION_KEY` (`openssl rand -base64 32`). Then:

```bash
docker compose pull && docker compose up -d
```

Open that URL (default [http://localhost:8087](http://localhost:8087)). Sign in as `admin` / `admin` and set a new password. The rest of the app stays locked until you do.

Trying it with sample SOPs: [Evaluating SOPandGO](docs/ops/evaluating.md). A real lab: [Deployment](docs/ops/deployment.md), including the [first-week checklist](docs/ops/deployment.md#first-week-checklist).

## Read next

- [Evaluating](docs/ops/evaluating.md) — a local trial with demo data
- [Deployment](docs/ops/deployment.md) — environment, mail, backups, and upgrades
- [Backup and restore](docs/ops/backup-and-restore.md)
- [Security model](docs/concepts/security-model.md)
- [Versioning and signatures](docs/concepts/versioning-and-signatures.md)
- [Contributing](CONTRIBUTING.md)
- [Changelog](CHANGELOG.md) — what each release added

## Disclaimer

This software is provided as-is, without warranty of any kind.

It is not validated for regulated environments and does not replace institutional, legal, or regulatory responsibilities. You are responsible for deciding whether it fits your lab.

## License

MIT License  
© 2025-2026 Leopold Dürrauer

This is an independent open-source project and is not affiliated with any employer or institution.

### Third-Party Assets

Demo data includes media used under other licenses. Credits and links are in `backend/demo/sops/87de1162-589b-4c91-8c37-8fae346a1ccc/assets/ATTRIBUTIONS.md`. The Gopher image is based on the Go Gopher created by Renee French, licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
