# Versioning and Signatures

This document describes how SOP versions and acknowledgments are handled in sopandgo,
and which guarantees the system provides.

It is intended for lab managers, technical staff, and auditors who want to understand
how SOP changes and confirmations are tracked.

---

## Core principles

sopandgo is built around the following principles:

- SOP **content** history is **append-only**
- Past version **content and assets** are **immutable**
- Acknowledgments are tied to **specific versions**
- Changes are **explicit and traceable**
- Publishing goes through an explicit **lifecycle** (draft → release candidate → published)

Convenience is deliberately traded for clarity and auditability.

---

## SOP versions

An SOP in sopandgo consists of one or more **versions**.

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
- `POST .../versions/{id}/reject` — body `{"reason"}` required (`sop:sign:approver`); **409** if not an RC
- `POST .../versions/{id}/add-reader` — reader acknowledgment (`sop:sign:reader`); **409** unless the version is `published`

Author acknowledgments are recorded when the version is created. Approver acknowledgments are recorded only by **approve**. There is no separate endpoint for either.

Only one RC per SOP may be open at a time. When a new version is published, the previously published version becomes `superseded`.

Only one version is **current** at a time: the published one. `GET /api/sops/{sopID}` names it in `published_version` and `published_version_id`, and the newest version of any state in `latest_version` (`id`, `version`, `status`, `created_at`); each is `null` until such a version exists. These are worked out from `sop_version_states` on every read, never stored on the SOP, so they always agree with the version list and with `GET .../version-latest`. The version page uses them to point from a draft, release candidate, rejected or superseded version to the current one, and to tell editors and approvers on the published version that a newer draft or release candidate is waiting.

The change summary is stored on the version and shown in the version list, on the version page, and in the dashboard’s recent-publishes list. The version page also shows a line diff against the previous published version (or the previous version, if nothing is published yet) on its **Changes** tab, beside the document. `?view=changes` opens that tab directly; an approver opening a release candidate lands on it.

When an email transport is switched on (not manual links), **publishing** a version emails other active users who can reader-sign. If mail cannot be delivered, the version stays published. Optional Slack / Gotify / generic webhook notifications for publish, RC, and reject are configured under admin Settings → Integrations and likewise never roll back the lifecycle action. Reminder digests for unsigned acknowledgments are not part of 1.0; use the dashboard and training coverage views.

---

## Immutability guarantees

After a version is created:

- SOP **content files** for that version are treated as read-only
- **Asset files** for that version are treated as read-only
- Content and asset **hashes** in the database are not rewritten
- **Lifecycle state** may change; each transition is appended to `sop_version_states` with actor and timestamp

The system records a content hash for each version. Whenever an SOP is retrieved, the system recalculates this hash. If files are modified outside the application, the mismatch is detected and flagged immediately.

This allows sopandgo to proactively identify integrity issues without silently serving corrupted content.

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

An acknowledgment in sopandgo means:

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

At any point, sopandgo can answer questions such as:
- Which version of an SOP was active (published) at a given time?
- Who promoted, approved, or rejected a version?
- Who acknowledged that version (and as author, approver, or reader)?
- When was the SOP last changed, and by whom?
- What changed between versions?

These answers are derived from:
- version content records and hashes
- lifecycle state history
- acknowledgment records
- tamper-evident audit logs (cryptographically chained)

No reconstruction or inference is required.

---

## Non-goals

sopandgo does not attempt to:
- automatically enforce SOP usage
- infer compliance status
- replace formal training systems
- satisfy regulatory validation requirements

It provides **traceability**, not certification.

---

## Summary

sopandgo treats SOPs as living documents with a permanent, inspectable history.

Versioning and acknowledgments are explicit by design, favoring clarity and
accountability over silent updates or implicit assumptions. Publishing is a
deliberate approve step after draft and release-candidate review.
