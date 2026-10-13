---
name: sop-lifecycle
description: >-
  Keep SOP version content, acknowledgments, archive, and the audit chain
  consistent. Use when changing drafts, release candidates, publish, reject,
  supersede, reader or approver signatures, archive, assets, PDF artifacts,
  or audit events.
---

# SOP lifecycle

`docs/concepts/versioning-and-signatures.md` is the source of truth for versions, acknowledgments, and archive. Read it, the concurrency and audit sections of `docs/dev/architecture.md`, and `backend/internal/sop/README.md` before changing this path. Audit writes follow `backend/internal/audit/README.md`.

## Content and status

- Markdown and assets are immutable after the version is created. A later edit is a new version.
- Status moves `draft` → `rc` → `published`, or `rc` → `rejected`. Publishing sets the previous published version to `superseded`.
- One release candidate per SOP. Creating a version does not publish it.
- The current published version and the latest version are derived from `sop_version_states` on every read. Do not store them on the SOP row.
- The author acknowledgment is recorded when the version is created. The approver acknowledgment is recorded only by approve. A reader acknowledgment applies only to a `published` version.
- Reject and archive reasons are required and at most 500 characters.

## Archive

Archive applies to the SOP, not to a version, and it deletes nothing. It is refused while a release candidate is open. While archived, new versions, promotions, approvals, rejections, reader signatures, and asset uploads return `409` with `{"error": "sop_archived"}`. Tags and favorites can still change. Archived SOPs leave the default list, signature status, and team coverage. Versions, signatures, PDFs, and assets stay readable.

## Transaction and audit

The service owns the transaction, the files, and the audit write.

- Call `audit.Log` inside the same transaction as the state change. If the audit write fails, roll the change back.
- Remove files written for a change that does not commit.
- A new event type gets a constant and an entry in `knownEventTypes` in `backend/internal/audit/types.go`. A new entity type gets the same treatment in `knownEntityTypes`. With `AUDIT_STRICT_TYPES=true`, unknown types are rejected.
- Give that event an English Paraglide message and a row in `EVENT_TITLES` in `frontend/src/lib/audit/present.ts`. Other locales fall back to English.
- The HTTP API only lists audit logs. There is no route that inserts, edits, or deletes them.

## After the commit

PDF rendering and Slack, Gotify, and webhook delivery run after the lifecycle commit. A failure is logged. Missing PDFs are retried by the backfill on the next start. Neither failure undoes create, promote, approve, or reject.

## Same change

Update `docs/concepts/versioning-and-signatures.md` and `backend/internal/sop/README.md` when a guarantee changes. A new route updates the handler and `frontend/src/lib/sdk/` together. A schema change is a new migration in `backend/internal/storage/migrations.go`; never edit a shipped one.
