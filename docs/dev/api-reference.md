# HTTP API surface

The product is the self-hosted web app. The Go HTTP API under `/api` is an **implementation detail** for that UI, not a supported public integration product. Prefer the concepts and ops docs for how the system behaves.

## Where to look

| Concern | Source of truth |
| --- | --- |
| Route registration (method + path) | [`backend/internal/api/api.go`](../../backend/internal/api/api.go) |
| Request/response shapes used by the UI | [`frontend/src/lib/sdk/`](../../frontend/src/lib/sdk/) |
| Auth, tokens, RBAC | [`docs/dev/auth-architecture.md`](./auth-architecture.md) |
| Lifecycle and acknowledgments | [`docs/concepts/versioning-and-signatures.md`](../concepts/versioning-and-signatures.md) |
| Operator deploy / mail / integrations | [`docs/ops/deployment.md`](../ops/deployment.md) |

When you add or change a route, update handlers and the frontend SDK in the same change. Do **not** maintain a separate endpoint encyclopedia unless you are deliberately supporting external HTTP clients.

## Calling the API

Paths below are Go listener paths (prefix `/api`).

- **Browser / SvelteKit:** talks to the backend on the internal loopback; the UI uses your app **`ORIGIN`**.
- **Direct HTTP** (scripts behind the bundled Caddy proxy): `{ORIGIN}/sopandgo/api/...` (Caddy strips `/sopandgo`). Example: `POST https://lab.example.org:8087/sopandgo/api/auth/login`.

Protected routes expect `Authorization: Bearer <access_token>`. Login/refresh return JSON tokens; the SvelteKit app stores them in HTTP-only cookies.

## Auth and scopes (summary)

- Public: `GET /api/health`, login, refresh, logout, reset-password completion.
- Everything else: bearer access token, plus optional `requireScope(...)` (see `api.go`).
- Bootstrap / `must_change_password`: only `GET /api/auth/me`, `PATCH /api/auth/me/update-password`, and `PATCH /api/auth/me/locale` until the password is changed.
- `GET /api/auth/me` includes `locale`. `PATCH /api/auth/me/locale` with `{ "locale": "<tag>" }` stores a tag from the supported list (`en`, `de` today) and rejects anything else with 400.
- `GET /api/admin/settings/email` includes `default_locale`. `PATCH /api/admin/settings/default-locale` with `{ "default_locale": "<tag>" }` sets the organization language used for shared notifications and PDF chrome (admin only).
- **Admin HTTP tools** (users, sessions, audit log UI APIs, integrity, mail, integrations, backups) require the `admin:integrity` scope (admin role). The `auditor` role has an `audit:read` scope in RBAC, but **no audit-log routes currently require it** — audit listing is admin-only. Auditors can still use the normal signed-in SOP read surfaces.

## Common status codes

| Code | Typical meaning |
| --- | --- |
| `400` | Bad input |
| `401` | Missing/invalid token |
| `403` | Missing scope or password-change gate |
| `404` | Missing resource |
| `409` | Invalid lifecycle or backup conflict |
| `412` | Encryption key / credentials precondition |
| `422` | Backup archive validation failed |
| `429` | Rate limit (strict on login/reset; general elsewhere) |
| `502` | Upstream mail/integration failure on test send |
| `503` | PDF disabled or backup maintenance lock |

Exact bodies and query parameters: read the handler and the SDK caller for that route.
