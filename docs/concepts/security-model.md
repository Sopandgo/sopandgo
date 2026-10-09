# Security Model

This document describes the security assumptions, guarantees, and limitations of
sopandgo.

It is intended to help users understand **what the system protects against**, **what
it does not**, and **how responsibility is shared** between the software and the
organization operating it.


## Scope and intent

sopandgo is designed as a **self-hosted internal tool** for managing SOPs within small
labs and research organizations.

Its security model focuses on:
- Preventing accidental or unauthorized access.
- Maintaining traceability of changes and acknowledgments.
- Allowing administrators to revoke access when team members leave.

It does **not** attempt to provide enterprise-grade security or replace institutional
IT infrastructure.


## Trust assumptions

sopandgo operates under the following assumptions:

- The host system (server, VM, or workstation) is trusted and maintained by the
  organization.
- Administrators are trusted users with broad authority.
- Network access to the service is intentionally limited (e.g. internal network,
  VPN, reverse proxy).

If these assumptions do not hold, sopandgo cannot provide meaningful protection.


## Identity and authentication

Users authenticate using locally managed credentials.

Key properties:
- **Invite-Only Registration:** Users are registered by admins, but passwords are set solely by the user via a secure email invite link. Admins never see or handle user passwords.
- **Bootstrap admin:** On first boot with an empty database, a default `admin` / `admin` account is created and **must change password before other application features unlock**.
- **Managed User Registration:** Users are created by admins.
- **Local Storage:** User identities and roles are stored in the local database.
- **Secure Hashing:** Passwords are stored using cryptographic hashing (Bcrypt).
- **Dual-Token System:** The system uses short-lived access tokens (5 minutes) and server-tracked refresh tokens. Refresh tokens can be revoked by an admin.
- **Password Reset Tokens:** Resets are handled via high-entropy, one-time-use tokens sent via email. These tokens are hashed in the database and deleted immediately upon use ("Burn-After-Reading").
- **Brute-Force Protection:** Login and Password Reset endpoints are protected by strict IP-based rate limiting to prevent credential guessing attacks.
- **Administrative Recovery:** Admins can trigger password resets.

External identity providers (SSO, LDAP, OAuth) are **out of scope**.


## Authorization and access control

Access to SOPs and administrative actions is controlled by **roles** and **scopes**.

- **Enforcement:** Authorization decisions are made server-side. Client-side state is never treated as authoritative.
- **Immediate account checks:** Deactivating a user or changing their role takes effect on the next API request. The server loads the account and rejects inactive users. It authorizes with the role stored in the database, not the role copied into the access token.
- **Auditability:** Every change to a user's role or status is logged in the audit trail.
- **Roles in practice:** `admin` has full tools (including audit log APIs). `approver` / `editor` / `viewer` cover the publish and read loop. `auditor` is a reserved role with SOP read access; the admin Audit Logs UI and APIs remain admin-only in 1.0.


## Session management

sopandgo uses **short-lived access tokens** combined with **server-tracked login sessions**.
The system uses Platform-Agnostic Security Tokens (PASETO) instead of JWT, avoids common JWT footguns.

This allows for:
- **Session revocation:** Revoking a refresh session takes effect immediately for new access tokens. An access token that was already issued stays valid until it expires (at most 5 minutes), because those tokens are not stored on the server.
- **Per-device sessions:** Each login is its own refresh session. A user can log out that session. An admin can revoke every session for one user, or trigger a system-wide panic logout. The admin session list does not include refresh tokens.
- **No Sensitive URLs:** No sensitive identifiers are embedded in SOP content/assets. One-time invite/reset tokens are delivered via email links and are time-boxed and single-use.

## Outbound integrations

Admins may configure instance-level destinations (Slack Incoming Webhooks, Gotify, generic HTTP webhooks) under Settings. Those destinations receive SOP lifecycle and selected ops alerts.

- **Admin-controlled only:** webhook URLs and tokens are stored encrypted in SQLite; the AES key stays in the environment (`SECRET_ENCRYPTION_KEY`). APIs never return decrypted secrets.
- **Trust the destination:** anyone who can administer the instance can point notifications at an arbitrary HTTPS endpoint. Treat that as equivalent to other admin powers (mail settings, user invites).
- **Best-effort delivery:** failed notification sends are audited (`notification_failed`) and do not undo publish/promote/reject.
- **Not a bot platform:** Incoming Webhooks / HTTP push only — no OAuth Slack apps, slash commands, or interactive components.


## Data integrity and auditability

sopandgo assumes a "Trust but Verify" approach to the filesystem. While the application logic is trusted, the underlying filesystem state is considered potentially volatile (e.g., bit-rot, manual file edits, or external sync errors).

sopandgo distinguishes between **content** and **metadata**:

- **Filesystem Storage:** SOP content and assets are stored as versioned files.
- **Database Metadata:** Metadata, acknowledgments, and audit events are stored in a database.
- **Integrity Verification:** For each SOP version, a content hash is recorded. The system can detect if files have been modified outside the application.
- **Tamper-Evident Audit Logs:** Audit logs are cryptographically linked using a SHA-256 hash chain. Any modification or deletion of a past log entry breaks the chain, making tampering detectable via built-in verification tools.
- **Proactive Verification of SOPs:** The Service Layer re-verifies SHA-256 hashes during retrieval of full SOP summaries and Audit Logs to ensure that what the user sees is exactly what was originally committed to disk.
- **On-Demand Checks:** Any signed-in user can re-check a single version or asset file from the SOP pages. The result is one of **verified** (hash matches), **corrupt** (hash differs), **missing** (file gone from disk), or **couldn't check** (the check itself failed). The check only answers for the SOP named in the request path; an asset or version from another SOP returns 404.
- **System Scan:** Admins can run a full scan at **Admin → Integrity** (`/admin/integrity`). It re-hashes every version and asset and verifies the audit hash chain. Each run is written to the audit log, and a failed run sends `integrity_check_failed` to enabled integrations. The scan runs only when an admin starts it; nothing schedules it.


## What sopandgo protects against

sopandgo is designed to reduce risk from:

- Accidental or unauthorized SOP modification.
- Missing or disputed Acknowledgment records
- Confusion regarding which SOP version was active at a specific time.
- Unauthorized viewing of internal SOPs
- Brute-force credential attacks.
- Undetected tampering of audit history, although external anchoring is not enforced by sopandgo yet.

## What sopandgo does NOT protect against

sopandgo does **not** protect against:

- A compromised host operating system on the server.
- Malicious administrators or users with direct file/database access, although hash chained audit logs make this detectable.
- Physical access to the server.
- Institutional network breaches or 'Man-in-the-Middle' attacks between the client and the server. While the internal Caddy instance handles routing, end-to-end encryption depends on the operator correctly configuring external SSL/TLS termination.

It is **not** a hardened security application.


## Operational responsibilities

Operators of sopandgo are responsible for:
- **Host Security:** Securing the OS and controlling physical/SSH access.
- **External SSL/TLS:** The container includes an internal Caddy instance for routing, an external reverse proxy is not included.
- **Data Persistence:** Operators must ensure the backend/data volume is correctly mounted to persistent storage. If the volume is lost, all audit logs and user identities are lost.
- **Backups:** Performing regular backups of both the filesystem data and the SQLite database.
- **Updates:** Applying software and security updates to the host and the container.


## Regulatory considerations

sopandgo is **not validated** for use in regulated environments such as GMP, GLP, or ISO-certified processes. While it provides traceability and audit-friendly records, responsibility for regulatory compliance remains entirely with the organization.