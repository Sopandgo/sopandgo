# Authentication & Authorization Architecture

## 1. Token Strategy: Stateless Access, Stateful Refresh

sopandgo uses a hybrid token model. This allows the backend to verify common requests (like viewing an SOP) without hitting the database, while still maintaining the ability to revoke access within minutes.

### Access Tokens (Stateless)

* **Protocol:** PASETO (Platform-Agnostic Security Tokens) v4.local.
* **Library:** `aidanwoods.dev/go-paseto` (V4 implementation).
* **Payload:** Includes `user_id`, `role`, and `exp` (expiry).
* **Duration:** 5 minutes.
* **Purpose:** Rapid authorization of API requests.

### Refresh Tokens (Stateful)

* **Type:** UUID v4 stored in the `refresh_tokens` table.
* **Duration:** 7 days.
* **Storage:** Managed by the client (SvelteKit) and passed in the JSON body of `/api/auth/refresh`.
* **Validation:** Every refresh request triggers a database join between `refresh_tokens` and `users` to verify that both the session and the user account are still active.

### Password Reset Tokens (One-Time Use)
**Type:** High-Entropy Random String (32 bytes from crypto/rand).

**Storage:** Only the **SHA-256 Hash** is stored in the DB (password_reset_tokens).

**Duration:** Time-boxed (24 h).

**Security:** Burn-After-Reading. The token is deleted immediately upon use to prevent replay attacks.

## 2. Password Security

sopandgo follows current industry standards for credential storage:

* **Algorithm:** Bcrypt (`golang.org/x/crypto/bcrypt`).
* **Cost Factor:** 12 (balancing resistance to brute-force with server performance).
* **Hashing Flow:** Passwords are hashed in the `internal/auth` package before ever reaching the database layer.
* **Password Reset:** Valid passwords can only be set by the user via the Password Reset Token flow. This ensures no admin ever knows a user's password. Only Admins can trigger Password Reset.


## 3. Revocation Logic

Revocation is handled at three levels of granularity. When a revocation event occurs, the server marks the corresponding Refresh Token(s) as `is_active = 0`.

| Action | API Endpoint | Logic |
| --- | --- | --- |
| **Session Logout** | `POST /api/auth/logout` | Deactivates the specific Refresh Token used. |
| **User Revocation** | `DELETE /api/admin/users/{userID}/sessions` | Deactivates all Refresh Tokens associated with a `userID`. |
| **Start Password Reset** | `POST /api/admin/users/{userID}/reset-password` | Admin starts invite/reset; email when a saved transport is switched on, otherwise a manual link (`effective_mail_mode`). |
| **Complete Password Reset** | `PATCH /api/auth/reset-password` | User sets a new password with the one-time token; related sessions are revoked as part of the reset flow. |
| **System Panic** | `DELETE /api/admin/sessions` | Deactivates every active Refresh Token in the database. |

> **Note:** Deactivation and role changes are enforced on the next API request, using the account stored in the database. Revoking a refresh session stops new access tokens immediately. An access token that was already issued stays valid until it expires (at most 5 minutes).


## 4. Middleware & RBAC

The Go backend uses a "Chain of Responsibility" for API protection:

1. **Rate limiting:**
    * **Strict limiter:** Protects `/api/auth/login` and password-reset completion (token bucket: burst 3; refill 1 req/12s) to limit brute-force attempts.
    * **General limiter:** Protects other rate-limited endpoints (token bucket: burst 10; refill 20 req/s).
    * **Proxy awareness:** Client IP for rate limits is `RemoteAddr`, unless that peer is the loopback proxy (Caddy on `127.0.0.1`). In that case the last `X-Forwarded-For` hop is used, because Caddy appends the address it observed. Client-supplied prefixes and `X-Real-Ip` are ignored.
2. **`withAuth`:** Extracts the PASETO from the `Authorization` header, verifies the signature, and injects the `user_id` and token `role` into the Request Context.
3. **`withPasswordChangeGuard`:** Loads the user. Inactive accounts are rejected. The context role is replaced with the role stored in the database, so a demotion applies before `requireScope` runs. Accounts flagged to change their password may only call `GET /api/auth/me`, `PATCH /api/auth/me/update-password`, `PATCH /api/auth/me/locale`, and `PATCH /api/auth/me/theme`.
4. **`requireScope`:** Checks the context role against a map of required permissions (Scopes). If the role lacks the necessary scope (e.g., a `viewer` trying to access `admin:integrity`), it returns `403 Forbidden`.


## 5. Audit Logging

The `audit_events` table is the system's Tamper-Evident "Black Box."

* **Format:** JSON payloads for flexibility.
* **Trigger: Automatic execution via the Service Layer.**
    * Unlike traditional logging, audit events occur inside the database transaction.
    * Atomicity: If the audit log fails to write, the entire operation (e.g., creating a user) is rolled back. This guarantees that no state change can occur without a corresponding log record.
* **Integrity:** The HTTP API only **lists** audit events (`GET /api/admin/audit-logs`). Writes happen inside service-layer transactions; there is no public `POST`/`UPDATE`/`DELETE` for logs.
* **Hash Chaining**:
    * Every log entry contains a cryptographic hash of itself and the previous entry's hash.
    * This creates an immutable chain.
    * The API automatically verifies this chain on retrieval to detect if logs have been modified or deleted manually.
    * Events written without a caller transaction (mail, notifications, logins, settings changes, startup, PDF artifacts) serialize per logger and take SQLite's write lock before reading the chain tip, so concurrent writers cannot link two entries to the same predecessor. If another writer holds the lock beyond `AUDIT_BUSY_TIMEOUT`, the failure is reported to the application log.


## 6. Database Schema (Security Relevant)

### `users` table

* `role_id`: String (default roles enforced via migration are: `"admin"`, `"approver"`, `"editor"`, `"viewer"`, and `"auditor"`).
  * **Note on `auditor`:** The role exists and is granted `audit:read` in the RBAC map, but audit-log HTTP endpoints currently require `admin:integrity`. In practice auditors get signed-in SOP **read** access (and a UI badge); they do **not** get the admin Audit Logs screens. Prefer `viewer` unless you are intentionally reserving the role for a future audit UI.
* `is_active`: Boolean (Integer 0/1). If 0, all authentication and refresh attempts fail.

### `refresh_tokens` table

* `token_id`: Primary Key (UUID).
* `is_active`: Boolean. Used for revocation.
* `expires_at`: Timestamp. Handled by a background cleanup task to prevent DB bloat.

### `password_reset_tokens` table

* `user_id`: Primary Key (Foreign Key to users). Enforces **Single Active Token** per user
* `token_hash`: TEXT (SHA-256 hash of the token). Raw tokens are never stored.
* `expires_at`: Timestamp.

### `audit_events` table
* `hash`: SHA-256 signature of the current record.
* `prev_hash`: The hash of the immediately preceding record, forming the integrity chain