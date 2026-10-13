# Versioning and Signatures

This document describes how SOP versions and acknowledgments are handled in SOPandGO,
and which guarantees the system provides.

It is intended for lab managers, technical staff, and auditors who want to understand
how SOP changes and confirmations are tracked.

---

## Core principles

SOPandGO is built around the following principles:

- SOP **content** history is **append-only**
- Past version **content and assets** are **immutable**
- Acknowledgments are tied to **specific versions**
- Changes are **explicit and traceable**
- Publishing goes through an explicit **lifecycle** (draft → release candidate → published)

Convenience is deliberately traded for clarity and auditability.

---

## SOP versions

An SOP in SOPandGO consists of one or more **versions**.

Each version holds a complete snapshot of the SOP content at a point in time.

A version includes:
- Structured SOP content (Markdown)
- Referenced assets (images, diagrams, attachments)
- Metadata (author, timestamp, change summary)
- A cryptographic content hash
- A **lifecycle status** (`draft`, `rc`, `published`, `rejected`, or `superseded`)

**Content** (Markdown file and assets) is never rewritten after creation. **Status** may change as the version moves through the lifecycle; those transitions are recorded in `sop_version_states`.

---

## Version lifecycle

Creating a version does **not** publish it. The flow is:

```text
draft  --promote-->  rc  --approve-->  published
                       |
                       +--reject-->  rejected
```

| State | Meaning |
| --- | --- |
| `draft` | Editable workflow state after create. Content on disk is already fixed; the version is not yet under review. |
| `rc` | Release candidate. Promoted by an editor (`sop:write`). Waiting for an approver. |
| `published` | Current official version. Approving an RC publishes it and records an **approver** acknowledgment. |
| `rejected` | RC declined by an approver (reason required). |
| `superseded` | A previously published version automatically moved aside when a newer version is published. |

API (scopes abbreviated):

- `POST /api/sops/{sopID}` — create a **draft** (`sop:write`); requires `change_summary`
- `POST .../versions/{id}/promote` — draft → RC (`sop:write`); **409** if not a draft
- `POST .../versions/{id}/approve` — RC → published (`sop:sign:approver`); returns `{"id"}` (approver ack)
- `POST .../versions/{id}/reject` — body `{"reason"}` required, at most 500 characters (`sop:sign:approver`); **400** if it is missing or too long, **409** if not an RC
- `POST .../versions/{id}/add-reader` — reader acknowledgment (`sop:sign:reader`); **409** unless the version is `published`

Author acknowledgments are recorded when the version is created. Approver acknowledgments are recorded only by **approve**. There is no separate endpoint for either.

The reject reason is stored on the `rejected` state row and in that transition's audit event, and `GET .../versions/{id}/summary` returns it as `rejection` (`reason`, `actor_user_id`, `actor_name`, `created_at`; `null` unless the version is rejected). The version page shows who rejected the version, when, and why. Rejections recorded before reasons were stored have `reason: null`; their reason went only to the integration notifications and cannot be recovered.

Only one RC per SOP may be open at a time. When a new version is published, the previously published version becomes `superseded`.

Only one version is **current** at a time: the published one. `GET /api/sops/{sopID}` names it in `published_version` and `published_version_id`, and the newest version of any state in `latest_version` (`id`, `version`, `status`, `created_at`); each is `null` until such a version exists. These are worked out from `sop_version_states` on every read, never stored on the SOP, so they always agree with the version list and with `GET .../version-latest`. The version page uses them to point from a draft, release candidate, rejected or superseded version to the current one, and to tell editors and approvers on the published version that a newer draft or release candidate is waiting. Each row in the SOP list (`/sops`) opens the latest version; a versions button next to the favorite star opens the SOP page (`/sops/{sop_id}`) with every version. The list shows the newest version's status as a badge when it is a draft or release candidate, or when nothing is published yet. A version rejected after a published one gets no badge there, because readers still see the published version. The SOP page lists the rejected version.

The change summary is stored on the version and shown in the version list, on the version page, and in the dashboard’s recent-publishes list. The version page also shows a line diff against the previous published version (or the previous version, if nothing is published yet) on its **Changes** tab, beside the document. `?view=changes` opens that tab directly; an approver opening a release candidate lands on it. Added and removed lines that embed an uploaded image (`![…](assets/…)`) show that image under the line. Assets belong to the SOP, file names are unique per SOP and files are never rewritten, so a removed line shows exactly the image the older version showed.

When an email transport is switched on (not manual links), **publishing** a version emails other active users who can reader-sign. If mail cannot be delivered, the version stays published. Optional Slack / Gotify / generic webhook notifications for publish, RC, and reject are configured under admin Settings → Integrations and likewise never roll back the lifecycle action. PDF export follows the same rule. Create, promote, approve and reject generate the PDF for the new state after the change is committed. If rendering fails (for example, Gotenberg is unreachable), the request still succeeds, the failure goes to the application log, and the PDF backfill at the next start creates the missing file. Until then, downloading that PDF returns **404**. Reminder digests for pending acknowledgments are not sent. Emails for a release candidate, a rejection, or an archive are not sent. The home dashboard lists each person's own pending signatures.

---

## Archiving an SOP

An SOP that is no longer used, such as a replaced procedure or retired equipment, can be **archived**. Archiving applies to the SOP as a whole, not to a version, and it deletes nothing.

- **Who:** approvers and admins (`sop:sign:approver`), the same people who decide what readers use. They archive from the SOP page and must give a reason (at most 500 characters).
- **Precondition:** no release candidate may be waiting for review. Approve or reject it first, so no review is left on a frozen SOP. Drafts may stay; they stay drafts.
- **Read-only:** while archived, the SOP takes no new versions, promotions, approvals, rejections, reader signatures or asset uploads. Those requests answer **409** with `{"error": "sop_archived"}`. Tags and favorites can still change, because they only help people find the SOP.
- **Out of the way:** the SOP list (`/sops`) shows SOPs in use by default; its **Active / Archived** switch (`?archived=only`) shows the archived ones with an "archived" badge. Archived SOPs leave the signature status (`GET /api/auth/me/signature-status`), so nobody is asked to sign them, and the team overview (`GET /api/training/coverage`). They leave the dashboard's favorites too. The per-SOP coverage (`GET /api/sops/{sopID}/training`) still answers, as history.
- **Still readable:** every version, its signatures, PDFs and assets stay available. The SOP page and its version pages show who archived the SOP, when and why. The recent-publishes list is history and is not filtered.
- **Unarchive:** an approver or admin can bring the SOP back from its page. It returns exactly as it was archived.

API:

- `POST /api/sops/{sopID}/archive` — body `{"reason"}` (`sop:sign:approver`); **204**, **400** `invalid_archive_reason`, **404**, or **409** if it is already archived or a release candidate is pending
- `DELETE /api/sops/{sopID}/archive` — unarchive (`sop:sign:approver`); **204**, or **409** if it is not archived
- `GET /api/sops?archived=only|all` — the default lists SOPs in use; each row carries `archived_at` (`null` while in use)
- `GET /api/sops/{sopID}` returns `archive` (`reason`, `actor_user_id`, `actor_name`, `archived_at`; `null` while in use)

The archive state lives on the `sops` row (`archived_at`, `archived_by`, `archive_reason`) and describes only the current archive. Each archive and unarchive is in the audit log as `sop_archived` (with the reason) or `sop_unarchived`, so earlier archives remain traceable.

---

## Immutability guarantees

After a version is created:

- SOP **content files** for that version are treated as read-only
- **Asset files** for that version are treated as read-only
- Content and asset **hashes** in the database are not rewritten
- **Lifecycle state** may change; each transition is appended to `sop_version_states` with actor and timestamp

The system records a content hash for each version. Whenever an SOP is retrieved, the system recalculates this hash. If files are modified outside the application, the mismatch is detected and flagged immediately.

This allows SOPandGO to proactively identify integrity issues without silently serving corrupted content.

Users can also re-check a version's checksum or an asset file from the version page without reloading it. A failed check distinguishes a **corrupt** file (hash mismatch) from a **missing** file and from a check that **could not run**. Admins can scan all versions, assets, and the audit chain at once from **Admin → Integrity**.

---

## Acknowledgments and signatures

Users do not sign an SOP in general.
They acknowledge **a specific SOP version**.

An acknowledgment records:
- the SOP version identifier
- the user who acknowledged it
- the timestamp
- the acknowledgment type: **`author`**, **`approver`**, or **`reader`**

Approving a release candidate both publishes the version and writes an **approver** acknowledgment in the same transaction.

If a new version of an SOP is published:
- previous acknowledgments remain valid **only** for the older version
- users must explicitly acknowledge the new published version if required

This avoids ambiguity about which procedure was acknowledged.

---

## Tracking signature status

The app compares each user’s acknowledgment history to the **currently published** SOP versions and exposes that as a list you can read from **`GET /api/auth/me/signature-status`**.

In the UI, the same breakdown appears on:

- the **home dashboard** (`/dashboard`) — summary, “action required” lists, and links to sign the latest version; see `docs/concepts/home-dashboard.md`
- **Your profile** (`/profile`) — full **Training / Signatures** card. Language, appearance, and password are on `/profile/settings`.

The categories are:

- **Up to Date:** The user has acknowledged the latest published version.
- **Action Required:** The user acknowledged an older version, but a new version has since been published.
- **Not Started:** The user has never acknowledged any version of a published SOP.

---

## What acknowledgments mean

An acknowledgment in SOPandGO means:

> “This user confirmed awareness of this exact SOP version at this time.”

It does **not** imply:
- regulatory compliance
- correctness of the SOP
- enforcement of behavior

Interpretation of acknowledgments remains the responsibility of the organization.

---

## User lifecycle and acknowledgments

User accounts may be disabled or removed from active use.

- Past acknowledgments are never deleted
- Historical records continue to reference the original user identity
- Disabled users cannot create new acknowledgments

This preserves historical traceability even as team members change.

---

## Auditability

At any point, SOPandGO can answer questions such as:
- Which version of an SOP was active (published) at a given time?
- Who promoted, approved, or rejected a version, and why was it rejected?
- Who acknowledged that version (and as author, approver, or reader)?
- When was the SOP last changed, and by whom?
- Who archived or unarchived an SOP, when, and why?
- What changed between versions?

These answers are derived from:
- version content records and hashes
- lifecycle state history
- acknowledgment records
- tamper-evident audit logs (cryptographically chained)

No reconstruction or inference is required.

---

## Non-goals

SOPandGO does not attempt to:
- automatically enforce SOP usage
- infer compliance status
- replace formal training systems
- satisfy regulatory validation requirements

It provides **traceability**, not certification.

---

## Summary

SOPandGO treats SOPs as living documents with a permanent, inspectable history.

Versioning and acknowledgments are explicit by design, favoring clarity and
accountability over silent updates or implicit assumptions. Publishing is a
deliberate approve step after draft and release-candidate review.
