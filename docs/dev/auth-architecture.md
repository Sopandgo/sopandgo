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

### Two-Factor Challenge Tokens

**Type:** PASETO v4.local with its own in-memory key, separate from the access-token key, so a challenge can never be used as an access token. Restarting the server invalidates pending challenges.

**Payload:** `user_id`, `purpose = two_factor_challenge`, `exp`.

**Duration:** 5 minutes.

**Purpose:** Proves only the password step of a sign-in on an account with 2FA. The SvelteKit app keeps it in an httpOnly, `SameSite=Strict` cookie scoped to `/login`.

**Single use:** Each challenge carries a random ID (`jti`). When the second factor succeeds, the ID is marked used (check and mark in one step, in memory until the challenge expires) before the session is created. A used challenge can neither open a second session nor send more email codes; the client gets `401 challenge_expired`, and a reuse is audited as `login_failed` (`challenge_already_used`). The ID is marked only after a correct code, so a typo does not cost the password step.

## 1a. Two-Factor Authentication

**Policy.** `app_settings.two_factor_policy` is the organization's required second factor. `auth.SecondFactorMethods(user, mailOn)` is the one place that decides what a user may use; both sign-in steps call it, so a policy change also applies to a sign-in already under way.

| Policy | Second step for users without an app | Users with an app | Admins without an app |
| --- | --- | --- | --- |
| `none` | none | `totp` (+ `recovery_code` for admins) | none |
| `email` | `email` | `totp`, `email` | must enroll (`must_enroll_two_factor`) |
| `totp` | must enroll | `totp` (+ `recovery_code` for admins) | must enroll |

Email is never offered to admins, and only while an email transport is switched on. A user who must enroll signs in with the password and is then limited to setting up an app (see `withPasswordChangeGuard`).

When a second factor applies, sign-in takes two requests:

1. `POST /api/auth/login` with the password. Instead of tokens, it returns `{ "two_factor_required": true, "challenge_token", "methods", "email_sent", "expires_in" }`. When email is the only method, the code is mailed right away (`email_sent`). When the policy needs an emailed code but email is off, login fails closed with `503 second_factor_unavailable` and `login_failed` (`second_factor_unavailable`), and the server log says how to fix it.
2. `POST /api/auth/login/two-factor` with `{ "challenge_token", "method", "code" }` returns the access and refresh tokens. `POST /api/auth/login/two-factor/email` with `{ "challenge_token" }` mails a code first.

Both steps sit behind the strict login limiter. Step 2 also has a per-account limiter (burst 5, refill one per minute). The login audit event records `second_factor`; a wrong code writes `login_failed` with `failure_reason: invalid_second_factor`.

**Factors**

* **TOTP:** RFC 6238 (SHA-1, 6 digits, 30 s, one step of drift either way), via `github.com/pquerna/otp`. The secret is sealed with `SECRET_ENCRYPTION_KEY` (AES-GCM). `last_used_step` rejects a replayed code, even in a race. A new authenticator stays pending (`pending_secret_enc`) until its first code is confirmed, so replacing one never leaves the account without a working factor.
* **Email code:** 6 digits, SHA-256 hash stored, 10 minutes, five attempts, one send per minute. Accepted only under the `email` policy, for non-admins; the service checks the policy again when it issues and when it verifies a code. A used, burned, or expired code keeps its row with an empty hash, so the one-minute cooldown counts from the last send no matter how that code ended.
* **Recovery codes:** Admins only. 10 codes (`XXXXX-XXXXX`, 50 bits each), SHA-256 hashes stored, deleted when used, when replaced, when 2FA is turned off, and when the account stops being an admin.

**Self-service** (`/profile/settings`): `GET /api/auth/me/two-factor` (status), `POST …/setup` (current password; returns the secret, `otpauth://` URI, and QR code once), `POST …/confirm` (first code; signs out the user's other sessions and returns recovery codes to admins), `DELETE /api/auth/me/two-factor` (current password; refused with `409 required_by_org` while the policy requires an app for this account. Under `email`, a normal user may remove their app and sign in with emailed codes), `POST …/recovery-codes` (admins; current password). Endpoints that check the password share the strict login limiter, and a wrong password writes `login_failed` (`invalid_current_password`).

**Admin:** `DELETE /api/admin/users/{userID}/two-factor` resets a normal user's 2FA and signs them out (`403 admin_target` for admins). `GET`/`PATCH /api/admin/settings/security` reads and sets `two_factor_policy` (and reports `encryption_key_configured` and `email_transport_on`). Any level above `none` needs the encryption key and an admin who already uses an app (`409 actor_needs_two_factor`); `email` also needs a switched-on email transport (`409 email_transport_off`). Changes are audited as `two_factor_policy_updated`.

**Break-glass:** `./sopandgo recover-admin --email <email>` in the container resets an admin's 2FA, password, active flag, and sessions in one transaction with its audit entry. It has no HTTP route. See `docs/concepts/security-model.md`.

**Key loss:** If `SECRET_ENCRYPTION_KEY` is missing or changed, stored secrets cannot be read. The code step answers `409 authenticator_unavailable` and the server log names the recovery: reset normal users from Admin → Users, and admins with `recover-admin`. Recovery codes, and email codes under the `email` policy, still work, because they do not depend on the key.

## 2. Password Security

sopandgo follows current industry standards for credential storage:

* **Algorithm:** Bcrypt (`golang.org/x/crypto/bcrypt`).
* **Cost Factor:** 12 (balancing resistance to brute-force with server performance).
* **Hashing Flow:** Passwords are hashed in the `internal/auth` package before ever reaching the database layer.
* **Password Reset:** Valid passwords can only be set by the user via the Password Reset Token flow. This ensures no admin ever knows a user's password. Only Admins can trigger Password Reset.
* **Signed-in password change:** `PATCH /api/auth/me/update-password` takes `current_password` and `new_password`. A wrong or missing current password returns `403`, writes a `login_failed` audit event (`failure_reason: invalid_current_password`), and leaves the password unchanged, so an unattended open session cannot take over the account. Reset links (`PATCH /api/auth/reset-password`) never ask for it, because the one-time token proves identity. The bootstrap admin enters `admin` as the current password on first sign-in.


## 3. Revocation Logic

Revocation is handled at three levels of granularity. When a revocation event occurs, the server marks the corresponding Refresh Token(s) as `is_active = 0`.

| Action | API Endpoint | Logic |
| --- | --- | --- |
| **Session Logout** | `POST /api/auth/logout` | Deactivates the specific Refresh Token used. |
| **Sign Out Other Devices** | `POST /api/auth/me/sessions/sign-out-others` | The signed-in user sends their own `refresh_token`; every other active Refresh Token of theirs is deactivated. Rejects a token that is not one of their active sessions. Audited as `logout` with `scope: other_sessions`. |
| **Change Password** | `PATCH /api/auth/me/update-password` | Requires the current password. Deactivates all of the user's Refresh Tokens; the UI then signs out this session too. |
| **User Revocation** | `DELETE /api/admin/users/{userID}/sessions` | Deactivates all Refresh Tokens associated with a `userID`. |
| **Start Password Reset** | `POST /api/admin/users/{userID}/reset-password` | Admin starts invite/reset; email when a saved transport is switched on, otherwise a manual link (`effective_mail_mode`). |
| **Complete Password Reset** | `PATCH /api/auth/reset-password` | User sets a new password with the one-time token; related sessions are revoked as part of the reset flow. |
| **System Panic** | `DELETE /api/admin/sessions` | Deactivates every active Refresh Token in the database. |
| **Confirm 2FA** | `POST /api/auth/me/two-factor/confirm` | Deactivates every other Refresh Token of the user (the caller's own `refresh_token` is kept). |
| **Reset 2FA** | `DELETE /api/admin/users/{userID}/two-factor` | Deactivates all Refresh Tokens of that normal user. Audited as `two_factor_reset`. |
| **Break-glass** | `./sopandgo recover-admin` (no HTTP route) | Deactivates all Refresh Tokens of that admin. Audited as `admin_break_glass_reset`. |

> **Note:** Deactivation and role changes are enforced on the next API request, using the account stored in the database. Revoking a refresh session stops new access tokens immediately. An access token that was already issued stays valid until it expires (at most 5 minutes).


## 4. Middleware & RBAC

The Go backend uses a "Chain of Responsibility" for API protection:

1. **Rate limiting:**
    * **Strict limiter:** Protects `/api/auth/login`, both two-factor sign-in steps, password-reset completion, the signed-in password change, and the 2FA settings endpoints that check the current password (token bucket: burst 3; refill 1 req/12s) to limit brute-force attempts.
    * **Per-account limiter:** The second sign-in step is also limited per user (burst 5; refill 1 req/min), so code guesses from many IPs still stop.
    * **General limiter:** Protects other rate-limited endpoints (token bucket: burst 10; refill 20 req/s).
    * **Proxy awareness:** Client IP for rate limits is `RemoteAddr`, unless that peer is the loopback proxy (Caddy on `127.0.0.1`). In that case the last `X-Forwarded-For` hop is used, because Caddy appends the address it observed. Client-supplied prefixes and `X-Real-Ip` are ignored.
2. **`withAuth`:** Extracts the PASETO from the `Authorization` header, verifies the signature, and injects the `user_id` and token `role` into the Request Context.
3. **`withPasswordChangeGuard`:** Loads the user. Inactive accounts are rejected. The context role is replaced with the role stored in the database, so a demotion applies before `requireScope` runs. Accounts flagged to change their password may only call `GET /api/auth/me`, `PATCH /api/auth/me/update-password`, `PATCH /api/auth/me/locale`, and `PATCH /api/auth/me/theme`. Avatar upload/delete is **not** on that allow-list. While the policy requires an app for an account that has none (`must_enroll_two_factor`), it may additionally call `GET /api/auth/me/two-factor` and `POST /api/auth/me/two-factor/setup` and `…/confirm`; everything else returns `403`. The SvelteKit hook sends both kinds of account to `/profile/settings`.
4. **`requireScope`:** Checks the context role against a map of required permissions (Scopes). If the role lacks the necessary scope (e.g., a `viewer` trying to access `admin:integrity`), it returns `403 Forbidden`.


## 4a. Profile pictures

Signed-in users manage their own picture from Account settings:

| Action | API | Notes |
| --- | --- | --- |
| Upload / replace | `POST /api/auth/me/avatar` | Multipart field `file` (JPEG, PNG, or WebP; max 5 MB). The server center-crops to a square, writes four JPEGs under `DATA_DIR/users/<userID>/` (`avatar-96.jpg`, `avatar-256.jpg`, `avatar-512.jpg`, `avatar-1024.jpg`), and stores `avatar_path` + `avatar_content_hash` (hash of the 1024px file). Audited as `user_avatar_updated`. |
| Remove | `DELETE /api/auth/me/avatar` | Clears columns and deletes the size files. Audited as `user_avatar_removed`. |
| Fetch | `GET /api/users/{userID}/avatar?size=sm\|md\|lg\|xl` | Any authenticated active user. Default `size=sm` (96). `md`→256, `lg`→512, `xl`→1024. Returns JPEG with `ETag` / `Cache-Control: private`. |

`GET /api/auth/me` and admin user lists include `has_avatar` and `avatar_content_hash` (for cache-busting). The on-disk path is not exposed. The SvelteKit UI crops in the browser, then uploads; images are served through `/api/users/avatar` so the httpOnly access token never goes to `<img>`.


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
* `avatar_path`: Nullable relative directory under `DATA_DIR` (e.g. `users/<id>`). Empty means no picture.
* `avatar_content_hash`: Nullable SHA-256 of the processed 1024×1024 JPEG (ETag / cache buster).

### `refresh_tokens` table

* `token_id`: Primary Key (UUID).
* `is_active`: Boolean. Used for revocation.
* `expires_at`: Timestamp. Handled by a background cleanup task to prevent DB bloat.

### `user_two_factor` table

* `user_id`: Primary Key (Foreign Key to users).
* `totp_secret_enc`: AES-GCM-sealed TOTP secret of the confirmed authenticator.
* `pending_secret_enc`: Sealed secret of an authenticator waiting for its first code.
* `confirmed_at`: Set when the first code is confirmed; 2FA is on while it is set.
* `last_used_step`: Last accepted TOTP time step (replay guard).

### `two_factor_recovery_codes` / `two_factor_email_codes` tables

* Only SHA-256 hashes of codes are stored. Recovery codes are one row each (admins only); email codes are one row per user with `expires_at` and `attempts`.

### `app_settings.two_factor_policy`

* `none` (default), `email`, or `totp`; validated in Go, and an unknown stored value counts as `none`. See the policy table above.

### `password_reset_tokens` table

* `user_id`: Primary Key (Foreign Key to users). Enforces **Single Active Token** per user
* `token_hash`: TEXT (SHA-256 hash of the token). Raw tokens are never stored.
* `expires_at`: Timestamp.

### `audit_events` table
* `hash`: SHA-256 signature of the current record.
* `prev_hash`: The hash of the immediately preceding record, forming the integrity chain