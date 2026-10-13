# Security Model

This document describes the security assumptions, guarantees, and limitations of
sopandgo.

It is intended to help users understand **what the system protects against**, **what
it does not**, and **how responsibility is shared** between the software and the
organization operating it.


## Scope and intent

SOPandGO is designed as a **self-hosted internal tool** for managing SOPs within small
labs and research organizations.

Its security model focuses on:
- Preventing accidental or unauthorized access.
- Maintaining traceability of changes and acknowledgments.
- Allowing administrators to revoke access when team members leave.

It does **not** attempt to provide enterprise-grade security or replace institutional
IT infrastructure.


## Trust assumptions

SOPandGO operates under the following assumptions:

- The host system (server, VM, or workstation) is trusted and maintained by the
  organization.
- Administrators are trusted users with broad authority.
- Network access to the service is intentionally limited (e.g. internal network,
  VPN, reverse proxy).

If these assumptions do not hold, SOPandGO cannot provide meaningful protection.


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
- **Password change needs the current password:** Changing the password from account settings asks for the current one. Someone who finds a session left open on a shared PC cannot use it to lock the owner out. Reset links do not ask for it, because the one-time token already proves identity.
- **Brute-Force Protection:** Login, Password Reset, and the signed-in password change are protected by strict IP-based rate limiting to prevent credential guessing attacks.
- **Administrative Recovery:** Admins can trigger password resets.
- **Two-factor authentication:** Each user can add an authenticator app (TOTP, RFC 6238). When a second factor applies, the server asks for a code after a correct password, before it creates a session. A code is accepted once; replaying it fails. One password step opens at most one session, even if more valid codes are presented. Setting up, replacing, or turning off 2FA needs the current password. The TOTP secret is sealed with `SECRET_ENCRYPTION_KEY`, so 2FA cannot be turned on without that key, and a leaked database or backup alone does not reveal the secrets.
- **Organization policy (required second factor):** An admin chooses one of three levels under **Settings → General → Security**:

  | Level | Users without an app | Users with an app | Admins | Email codes |
  | --- | --- | --- | --- | --- |
  | **None** (default) | Password only | App code | App code if they set one up | Never |
  | **Email code or app** | Emailed code at every sign-in | App code or emailed code | Must set up an app | Normal users only |
  | **Authenticator app** | Must set up an app first | App code | Must set up an app | Never |

  Anyone can add an app voluntarily at any level. Only an admin who already uses an app can raise the level above none. The email level can only be chosen while an email transport is switched on. Accounts that must set up an app can only do that (plus change language, theme, or password) until they have one. Every change is audited as `two_factor_policy_updated`.
- **Email codes:** A 6-digit code mailed to the account's address (valid 10 minutes, one use, five tries, at most one send per minute, counted from the last send whether or not that code was used). The invite or reset link already proved that the user controls that mailbox, so no setup is needed. At the email level the second factor is "authenticator app **or** mailbox": someone with the password and access to the user's mailbox can sign in. Labs that do not want that choose the app level.
- **Why admins always need an app:** an admin's mailbox is the most valuable one to an attacker, the bootstrap admin has no real address, and an admin who signs in with an app can still change the policy if mail breaks.
- **If email stops working at the email level:** users without an app cannot sign in (the server refuses rather than letting the password alone through) until email works again or an admin lowers the level. Settings → Email warns about this before you switch email off.
- **Recovery codes (admins only):** Admins get 10 single-use recovery codes when they set up an app. Only SHA-256 hashes are stored. Codes stop working when the account is no longer an admin.
- **Admin reset for normal users:** Admins can reset a normal user's 2FA from **Admin → Users**; the user is signed out and sets it up again. Admin accounts cannot be reset there, so a stolen admin session cannot quietly strip another admin's second factor through the UI.
- **Rate limits on the second factor:** The code step uses the strict per-IP limiter plus a per-account limit, so guesses spread over many addresses still stop.

External identity providers (SSO, LDAP, OAuth) are **out of scope**.


## Break-glass recovery for admins

An admin who lost their password, authenticator, and recovery codes is recovered on the host, not in the web app. The operator opens a shell in the `sopandgo` container (for example the Coolify **Terminal**, or `docker compose exec`) and runs `./sopandgo recover-admin --email <email>`. In one transaction it removes the admin's second factor and recovery codes, sets a temporary password that must be changed at the next sign-in, reactivates the account, and signs out every session. Steps are in [deployment.md](../ops/deployment.md#locked-out-admin-break-glass).

- **No web route.** The command is part of the backend binary and has no HTTP endpoint. Nobody can reach it over the network, with or without an admin session.
- **Always audited.** Each reset writes `admin_break_glass_reset` in the same transaction as the change; if that audit write fails, nothing changes. Refused or cancelled runs write `admin_break_glass_failed`. Both carry the host name and OS user and enter the hash chain like every other event. The temporary password is never logged. Integrations subscribed to `break_glass_used` get an alert.
- **Admin accounts only.** Normal users are reset from **Admin → Users**.
- **Who can use it:** anyone with a shell in the container or write access to the data directory. That is already outside what SOPandGO defends (see below). The image runs Caddy, the SvelteKit server, and the Go API in one container, so code execution in any of them counts as host access: it could read `app.db` directly even without this command. Break-glass makes recovery possible and visible; it does not add a new boundary.


## Authorization and access control

Access to SOPs and administrative actions is controlled by **roles** and **scopes**.

- **Enforcement:** Authorization decisions are made server-side. Client-side state is never treated as authoritative.
- **Immediate account checks:** Deactivating a user or changing their role takes effect on the next API request. The server loads the account and rejects inactive users. It authorizes with the role stored in the database, not the role copied into the access token.
- **Auditability:** Every change to a user's role or status is logged in the audit trail.
- **Roles in practice:** `admin` has full tools (including audit log APIs). `approver` / `editor` / `viewer` cover the publish and read loop. `auditor` is a reserved role with SOP read access; the admin Audit Logs UI and APIs remain admin-only in 1.0.
- **Archiving SOPs:** Only approvers and admins (`sop:sign:approver`) archive or unarchive an SOP, and archiving requires a reason that goes into the audit log. Archiving hides an SOP and blocks changes to it on the server; it deletes no content, version, signature or audit record, and any signed-in user can still read the SOP. It is not a way to withdraw information from people who could already read it.
- **Profile pictures:** Each signed-in user may upload or remove their own picture. Uploads are type-checked (JPEG/PNG/WebP), size-capped (5 MB), and re-encoded to small JPEGs on disk. Any authenticated user can fetch another user's processed avatar the same way they can see display names in acknowledgment lists. Pictures are **not** a confidentiality boundary beyond the existing signed-in access model.


## Session management

SOPandGO uses **short-lived access tokens** combined with **server-tracked login sessions**.
The system uses Platform-Agnostic Security Tokens (PASETO) instead of JWT, avoids common JWT footguns.

This allows for:
- **Session revocation:** Revoking a refresh session takes effect immediately for new access tokens. An access token that was already issued stays valid until it expires (at most 5 minutes), because those tokens are not stored on the server.
- **Per-device sessions:** Each login is its own refresh session. A user can log out that session, or sign out every other device from account settings. An admin can revoke every session for one user, or trigger a system-wide panic logout. The admin session list does not include refresh tokens.
- **No Sensitive URLs:** No sensitive identifiers are embedded in SOP content/assets. One-time invite/reset tokens are delivered via email links and are time-boxed and single-use.

## Outbound integrations

Admins may configure instance-level destinations (Slack Incoming Webhooks, Gotify, generic HTTP webhooks) under Settings. Those destinations receive SOP lifecycle and selected ops alerts.

- **Admin-controlled only:** webhook URLs and tokens are stored encrypted in SQLite; the AES key stays in the environment (`SECRET_ENCRYPTION_KEY`). APIs never return decrypted secrets.
- **Trust the destination:** anyone who can administer the instance can point notifications at an arbitrary HTTPS endpoint. Treat that as equivalent to other admin powers (mail settings, user invites).
- **Best-effort delivery:** failed notification sends are audited (`notification_failed`) and do not undo publish/promote/reject.
- **Not a bot platform:** Incoming Webhooks / HTTP push only — no OAuth Slack apps, slash commands, or interactive components.

## Branding

Admins can set the organization's name, logo, and primary colour under **Settings → Branding**.

- **Public by design:** the sign-in page shows the name and logo, so `GET /api/branding` and `GET /api/branding/logo` need no session. Anyone who can reach the server can read the name, the colours, and the logo. Do not put anything confidential in them.
- **Images are re-encoded:** uploads must be PNG, JPEG, or WebP (at most 2 MB and 4096 pixels per side). The server decodes them and stores freshly encoded PNGs, so metadata and any non-pixel bytes are dropped. SVG is refused because it can carry script and the logo is served without a session. Responses carry `X-Content-Type-Options: nosniff`.
- **Readable colours:** a custom primary colour must reach 4.5:1 contrast against the theme's background; the server rejects anything else.
- **Name in emails:** the name replaces "SOPandGO" in invite, reset, sign-in code, and test messages. Line breaks and other control characters are rejected, so the name cannot add mail headers. A misleading name is an admin's choice, like any other admin power.
- **Attribution stays:** "Powered by SOPandGO" remains in the footer and sidebar.
- **Audited:** changes are logged as `branding_updated`, `branding_logo_updated`, `branding_logo_removed`, and `branding_reset` (back to every default; the event keeps the old values).

## Scheduled S3 backups

Admins configure automatic backups to an S3 bucket under **Settings → Backup**. Each run uploads the same full archive as **Export**: the database (including password hashes and the audit log) and every SOP file.

- **Admin-controlled only:** the settings live in SQLite; the secret access key is encrypted with `SECRET_ENCRYPTION_KEY` and never returned. Leaving the credentials empty uses the server's own AWS credentials (for example an IAM role) instead.
- **Trust the destination:** an admin can point backups at any bucket, which sends a full copy of the data there on every run. Admins can already download the same archive with **Export**, so this is not a new power, but it is persistent and quiet. Every change is audited as `backup_s3_settings_updated` (without the secret); review it with the other admin events.
- **Restores do not redirect backups:** a staged restore keeps the running instance's S3 settings instead of the archive's, so restoring an old or foreign archive cannot change where backups go.


## Data integrity and auditability

SOPandGO assumes a "Trust but Verify" approach to the filesystem. While the application logic is trusted, the underlying filesystem state is considered potentially volatile (e.g., bit-rot, manual file edits, or external sync errors).

SOPandGO distinguishes between **content** and **metadata**:

- **Filesystem Storage:** SOP content and assets are stored as versioned files.
- **Database Metadata:** Metadata, acknowledgments, and audit events are stored in a database.
- **Integrity Verification:** For each SOP version, a content hash is recorded. The system can detect if files have been modified outside the application.
- **Tamper-Evident Audit Logs:** Audit logs are cryptographically linked using a SHA-256 hash chain. Any modification or deletion of a past log entry breaks the chain, making tampering detectable via built-in verification tools.
- **Proactive Verification of SOPs:** The Service Layer re-verifies SHA-256 hashes during retrieval of full SOP summaries and Audit Logs to ensure that what the user sees is exactly what was originally committed to disk.
- **On-Demand Checks:** Any signed-in user can re-check a single version or asset file from the SOP pages. The result is one of **verified** (hash matches), **hash mismatch** (hash differs), **missing** (file gone from disk), or **couldn't check** (the check itself failed). The check only answers for the SOP named in the request path; an asset or version from another SOP returns 404.
- **System Scan:** Admins can run a full scan at **Admin → Integrity** (`/admin/integrity`). It re-hashes every version and asset and verifies the audit hash chain. Each run is written to the audit log, and a failed run sends `integrity_check_failed` to enabled integrations. The scan runs only when an admin starts it; nothing schedules it.


## What SOPandGO protects against

SOPandGO is designed to reduce risk from:

- Accidental or unauthorized SOP modification.
- Missing or disputed Acknowledgment records
- Confusion regarding which SOP version was active at a specific time.
- Unauthorized viewing of internal SOPs
- Brute-force credential attacks.
- Undetected tampering of audit history, although external anchoring is not enforced by SOPandGO yet.

## What SOPandGO does NOT protect against

SOPandGO does **not** protect against:

- A compromised host operating system on the server.
- Malicious administrators or users with direct file/database access, although hash chained audit logs make this detectable.
- Someone with a shell in the container resetting an admin's sign-in with `recover-admin`. This is the intended recovery path; it is audited and alerted, not prevented.
- Physical access to the server.
- Institutional network breaches or 'Man-in-the-Middle' attacks between the client and the server. While the internal Caddy instance handles routing, end-to-end encryption depends on the operator correctly configuring external SSL/TLS termination.

It is **not** a hardened security application.


## Operational responsibilities

Operators of SOPandGO are responsible for:
- **Host Security:** Securing the OS and controlling physical/SSH access.
- **External SSL/TLS:** The container includes an internal Caddy instance for routing, an external reverse proxy is not included.
- **Data Persistence:** Operators must ensure the backend/data volume is correctly mounted to persistent storage. If the volume is lost, all audit logs and user identities are lost.
- **Backups:** Performing regular backups of both the filesystem data and the SQLite database.
- **Updates:** Applying software and security updates to the host and the container.


## Regulatory considerations

SOPandGO is **not validated** for use in regulated environments such as GMP, GLP, or ISO-certified processes. While it provides traceability and audit-friendly records, responsibility for regulatory compliance remains entirely with the organization.