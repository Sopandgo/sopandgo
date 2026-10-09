# Architecture Overview

**sopandgo** employs a **Layered Architecture** to ensure separation of concerns, testability, and maintainability.

## System Layers

The backend is strictly divided into three distinct layers:

1. **Transport Layer (`api` package):**
  * **Role:** Handles HTTP requests, parses JSON, and performs basic input validation.
  * **Behavior:** "Dumb" logic-wise, but Security-Smart. It enforces IP-based Rate Limiting (Token Bucket) and handles Proxy IP Resolution before calling the Service layer.


2. **Service Layer (`auth`, `sop`, `audit`, `mail`, `notify`, `backup` packages):**
  * **Role:** The core brain of the application.
  * **Behavior:** Manages business rules, executes atomic database transactions, enforces concurrency locks, and calculates cryptographic integrity hashes.


3. **Data Access Layer (Private `store` functions):**
  * **Role:** Raw SQL execution.
  * **Behavior:** Completely hidden (private) within the service packages. It cannot be accessed directly by the API.

## Domain Model

The system is built around these core entities:

* **User:** A registered identity with an assigned role, status (RBAC), and `locale` (BCP 47 tag, default `en`).
* **Session:** A server-tracked record of an active login (Refresh Token).
* **Password Reset Token:** A hashed, time-boxed, single-use token used for user invites and password recovery.
* **SOP & Version:** An SOP is a logical container; a Version holds append-only Markdown content plus a lifecycle status (`draft` / `rc` / `published` / `rejected` / `superseded`).
* **Tag:** Global label attachable to SOP containers; admins can deactivate tags.
* **SOP favorite:** A private link between a **User** and an **SOP** (`sop_favorites` in SQLite). Used for bookmarks and list ordering; not shared between users.
* **Asset:** Immutable files (images/PDFs) linked to specific SOPs.
* **Audit Event:** A cryptographically linked, permanent record of a state change or significant system action.
* **Mail settings:** `smtp_settings` holds provider host, port, username, from-address, and AES-GCM-encrypted password. Resend API key is stored similarly when that transport is selected. `app_settings` holds `mail_mode` (`smtp` or `manual_links`), `mail_transport` (`smtp` or `resend`), and `default_locale` (organization language for shared notifications, test messages, and PDF chrome). The encryption key is supplied only via `SMTP_SECRET_ENCRYPTION_KEY`; configuration is edited in the admin UI. `users.locale` and `app_settings.default_locale` are plain text with no SQL allow-list, so a new language does not need a migration. Both columns are in SQLite and travel with backups.
* **Appearance:** `users.theme` is `light`, `dark`, or `system` (the default). Light uses the corporate theme and dark uses business. System follows the browser color scheme. The column is plain text with no SQL allow-list and travels with backups. Before sign-in, the navbar can set light or dark in a non-httpOnly `theme` cookie. With no cookie, appearance follows the system color scheme. After sign-in, `users.theme` applies and the cookie is ignored. Account language, appearance, and password live at `/profile/settings`. `/profile` shows account details and training signatures.
* **Integration settings:** `integration_settings` holds Slack Incoming Webhook URL (encrypted), Gotify URL + token (token encrypted), and generic webhook URL + optional bearer (encrypted), plus per-channel enable flags and event subscriptions. Same encryption key as mail. The `notify` package fans out after lifecycle commits (publish / RC / reject) and for ops alerts (S3 backup failure, failed integrity check).
* **Backup:** Admin export/validate/staged restore; optional scheduled S3 uploads (`BACKUP_S3_*`).

## Routing & Proxy Model
To simplify the frontend's communication with the backend, we use a Reverse Proxy within the container (Caddy):

* **Traffic Routing:** An internal **Caddy** instance orchestrates traffic:
  * `/sopandgo/*`  Go Backend (Port 8080)
  * `/*`  SvelteKit Frontend (Port 3000)

* **Storage:** A hybrid model using **SQLite** for relational metadata and the **Local Filesystem** for versioned content (Markdown/Blobs). This also ensures that backups only require to copy the data folder.
  

## Concurrency & Safety Model

The backend guarantees data consistency through a strict **Concurrency Model**:

* **Mutex Locking:** Each Service (`auth`, `sop`) utilizes internal `sync.RWMutex` locks.
* **Writes:** Serialized (one at a time) to prevent race conditions (e.g., creating two "Version 1"s simultaneously).
* **Reads:** Concurrent (`RLock`) to allow high performance during heavy load.
* **Atomic Transactions:** All complex operations (e.g., "Create SOP" or "Register User") are wrapped in a SQL Transaction.
* **Rule:** If the Audit Log fails to write, the Data change is rolled back.


## Identity & Security Model

The architecture uses a **Stateless/Stateful Hybrid**:

* **Invite-Only Flow:** Users are created with "unknowable" random passwords. Access is granted solely via email invites containing a secure token.
* **Stateless Access:** PASETO tokens are verified by the API without database lookups for speed.
* **Stateful Control:** Refresh sessions can be revoked immediately. Deactivation and role changes apply on the next API request. An access token already issued remains valid for at most 5 minutes after a session revoke.
* **Brute Force Protection:** The API layer employs a Rate Limiter with strict policies for Auth endpoints and relaxed policies for general API usage.

## Storage Model

* **Database (SQLite):** Stores relational data—user profiles, session status, audit logs, SOP metadata, tags, acknowledgments, and **`sop_favorites`** (user ↔ SOP bookmarks with composite primary key).
* **Filesystem:** Stores raw Markdown files and binary assets.

## Languages

The interface ships in English (`en`, the fallback), German (`de`), French (`fr`), Spanish (`es`), Portuguese (`pt`), Chinese (`zh`), Italian (`it`), Dutch (`nl`), Polish (`pl`), Japanese (`ja`), Korean (`ko`), Turkish (`tr`), and Swedish (`sv`). Signed-in pages use `users.locale`. Public pages set the locale cookie from the navbar language menu; resolution is that cookie, then `Accept-Language`, then English. Emails use the recipient's locale. Slack, Gotify, webhooks, and PDF chrome use `app_settings.default_locale`, because those outputs are shared. A PDF is generated once and is not re-rendered per reader.

SOP titles, Markdown, change summaries, reject reasons, tag names, and display names are stored as written and are not translated. Only the surrounding system sentences change.

Supported tags live in [`i18n/supported-locales.json`](../../i18n/supported-locales.json). The same list is embedded for the Go catalogs and must match Paraglide's `locales` in `frontend/project.inlang/settings.json`. Handlers reject any other tag. There is no locale prefix in URLs.

To add a language:

1. Append the BCP 47 tag to `i18n/supported-locales.json` and to the copy at `backend/internal/i18n/supported-locales.json` (a test fails if they differ), and to Paraglide's `locales`.
2. Add `frontend/messages/<tag>.json` and `backend/internal/i18n/messages/<tag>.json`.
3. Leave missing keys to fall back to English. Language pickers read the supported list and label each language in its own name. `lang` and `dir` on the document follow the active locale.

## PDF generation (backend)

When PDF export is enabled, the backend may convert Markdown to HTML and run it through **`bluemonday`** (HTML sanitizer) before sending markup to the PDF renderer (typically Gotenberg). That dependency may pull in smaller **indirect** packages (e.g. CSS parsing) via Go modules; they are expected and do not change the app’s public surface.

## Integrity Model

Integrity is verified via SHA-256 cryptographic hashes in two distinct ways:

1. **Content Integrity:** SOP versions and assets are hashed upon creation. This is automatically checked when retrieving a SOP version summary to detect disk tampering.

2. **Audit Integrity:** Audit logs are formed into a Hash Chain. The backend verifies the chain continuity on read to detect if logs have been deleted or altered.

3. **On-Demand Checks:** `GET /api/sops/{sopID}/versions/{versionID}/integrity` and `GET /api/sops/{sopID}/assets/{assetID}/integrity` re-hash one file. They return `{"hash_valid": bool}`, or a JSON error: 404 `version_not_found` / `asset_not_found` when the record does not exist or belongs to another SOP, 404 `file_missing` when the file is gone from disk, and 500 `integrity_check_failed` otherwise. `GET /api/admin/integrity` scans everything, audits the run, and notifies on failure.

## Audit Logging

All state-changing events (logins, password updates, role changes) are recorded.

* **Transactional:** State-changing actions commit their audit event in the same SQL transaction. Writes with no caller transaction (login, mail, notifications, startup, PDF artifacts) serialize per logger and take SQLite's write lock before reading the chain tip, so concurrent writers cannot fork the chain. Lock waits are bounded by `AUDIT_BUSY_TIMEOUT`; failures are reported to the application log.
* **Immutable:** The API prevents modification or deletion of logs.
* **Tamper-Evident:** Logs are cryptographically linked. Any modification breaks the hash chain, which the system can detect and flag.
* **Canonical Taxonomy:** `event_type` and `entity_type` are normalized to lowercase to keep filtering/reporting stable over time.
* **Optional Strict Mode:** With `AUDIT_STRICT_TYPES=true`, only known canonical audit types are accepted.


## SDK (Frontend)

The frontend communicates with the Go backend through a custom **TypeScript SDK**. This SDK centralizes authentication, error handling, and type safety, ensuring that the SvelteKit application remains decoupled from raw fetch calls.

### SDK Structure

The SDK is organized into domain-specific modules that mirror the backend service layer:

* **`auth`**: Handles login, logout, password updates, and session refreshes (sets HTTP-only cookies in SvelteKit).
* **`sops`**: Manages SOP containers, versions (including promote / approve / reject), summaries, acknowledgments, PDF download proxy, and **per-user favorites** (`favorite` / `unfavorite`, list filters `favorites_only` / `favorites_first`).
* **`tags`**: Creates/lists tags and attaches/detaches them on SOPs.
* **`assets`**: Handles binary file uploads, metadata retrieval, and integrity checks.
* **`admin`**: High-privileged operations including user management, session revocation, **audit log listing**, global integrity scans, mail settings, outbound integrations, and backup export/validate/staged restore (`/api/admin/backups/*`).

### Core Mechanisms

#### 1. Unified Client (`client.ts`)

All modules utilize a central `createClient` factory. This client automatically handles:

* **Base URL Injection**: Prepends the backend API path to all requests.
* **Authentication**: Automatically injects the `access_token` from cookies into the `Authorization` header for authenticated requests.
* **Content-Type Management**: Automatically stringifies JSON bodies and manages `FormData` boundaries for file uploads.
* **Error Guarding**: Catches connection failures and throws standardized `SERVICE_UNAVAILABLE` errors.

#### 2. Session Management (`auth.ts`)

Authentication state is managed via **HTTP-only cookies** to prevent XSS-based token theft:

* **Access Tokens**: Short-lived (5 minutes) for stateless API access.
* **Refresh Tokens**: Long-lived (7 days) for stateful session persistence.
* **Security Defaults**: Cookies are configured with `httpOnly`, `sameSite: 'lax'`, and dynamic `secure` flags based on the connection protocol.

### Usage Pattern

The SDK is typically initialized in SvelteKit `hooks.server.ts` or `+page.server.ts` using the `createSDK` factory:

```typescript
// Example: Fetching an SOP Summary in a SvelteKit Loader
export const load = async ({ locals, params }) => {
    try {
        const summary = await locals.api.sops.getVersionSummary(params.id);
        return { summary };
    } catch (err) {
        // Handle standardized SDK errors (e.g., FETCH_SUMMARY_FAILED)
        throw error(500, "Could not load document");
    }
};

```

### Home dashboard (signed-in landing)

The route **`/dashboard`** is the default post-login destination in the SvelteKit app. It loads signature status and a favorites slice from the SDK, surfaces **All SOPs** (`/sops`), and deep-links pending acknowledgments to **`/sops/{id}/v/latest`**. Conceptual overview: `docs/concepts/home-dashboard.md`.

### Draft editor and preview (new SOP version)

The route **`/sops/[sop_id]/new`** composes:

* **`SopVersionEditor`** — publish form, required change summary, markdown textarea, and a **Preview** tab.
* **`SopEditorAssetSidebar`** — **Import from Word** (`.docx` → markdown, when the draft is empty), asset list, and upload.

Draft markdown is **`bindable`** from the page into **both** components so Word import and the textarea always share one source of truth. Submitting the editor creates a **draft** version (`createVersion`); promote and approve on the version page complete publishing.

**Preview** uses the same **`MarkdownRenderer`** component as the immutable version viewer: identical markdown pipeline, `assets/…` resolution against the loaded asset list, and **`sanitizeSopHtml`** before `{@html}`. The UI uses DaisyUI **tabs** (single visible pane) so preview is not squeezed beside the source inside the main-column card.

### Integrity Integration

The SDK provides direct access to the system's **Integrity Model**:

* **Automatic**: The `sops.getVersionSummary` method (and admin audit list responses) return `hash_valid` as verified by the Go backend.
* **On-Demand**: `sops.checkIntegrity(sopId, versionId)` and `assets.checkIntegrity(sopId, assetId)` re-check one file and throw `SdkHttpError` on failure; `admin.checkIntegrity()` runs the system scan.
* **UI**: Pages call these through SvelteKit form actions, not a JSON proxy. `$lib/server/integrityActions` provides `verifyAsset` and `verifyVersion` for the `/sops/[sop_id]` pages, and `$lib/integrity` maps results to `verified` / `corrupt` / `missing` / `unavailable`. `IntegrityCheck.svelte` posts to those actions with `use:enhance` and also works without JavaScript. `/admin/integrity` runs the system scan through its own `?/run` action.

---
