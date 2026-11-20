# Errreport Package

Sends the API's unexpected errors to Sentry or GlitchTip (GlitchTip accepts the Sentry protocol), when an admin has turned that on under **Settings → Error reporting**. It is off by default. Why it is configured in the admin UI and why the DSN is public: [`docs/adr/0004-error-reporting-configured-by-admins.md`](../../../docs/adr/0004-error-reporting-configured-by-admins.md).

Operator docs: `docs/ops/deployment.md` (Error reporting). What leaves the instance: `docs/concepts/security-model.md` (Error reporting). HTTP surface: `/api/admin/settings/error-reporting*` and the public `GET /api/error-reporting/config` in `backend/internal/api/handle_error_reporting.go`.

## Pieces

| File | Role |
| --- | --- |
| **`service.go`** | `Service`: `Get`, `Save` (validates, stores, swaps the running client without a restart), `PublicConfig`, `Close` |
| **`reporter.go`** | The Sentry client (own `Hub`, so nothing global), the `scrub` applied to every event, and `Middleware` |
| **`testevent.go`** | `SendTest`: one message event, sent synchronously, so the admin learns whether the DSN works |
| **`validate.go`** | DSN (`http` or `https`, key, host, project) and environment rules, `DSNTarget` for the audit log |
| **`store.go`** | The `error_reporting_settings` row (migration 32). No row means off |
| **`carry.go`** | Keeps this instance's row across a backup restore (`WriteCarryOver` stage hook, `ApplyCarriedSettings` at startup) |

## What the middleware does

`api.Server.SetErrorReporting` wraps the whole handler, the demo guard included.

- **Panics** are always recovered, reporting on or off: logged with a stack trace, answered `500 {"error":"internal"}` if nothing was written yet, and reported. `http.ErrAbortHandler` is re-panicked, as net/http expects.
- **5xx responses** are reported as `HTTP <status> <METHOD> <pattern>`, grouped by method, route and status. **502** (a service the admin configured failed: SMTP, S3, Slack, the TSA, every test button) and **503** (maintenance lock, second factor unavailable) are not reported: they are the admin's to fix, not bugs.

## What an event contains

Error type and message, stack trace, the route pattern (`GET /api/sops/{sopID}`, never the URL), method, status, release (`APP_VERSION`), environment, and the Go runtime and OS. `server_name` is always `sopandgo`.

`scrub` removes, whatever an integration attached: the request (URL, query, headers, cookies, body, client IP), the user, breadcrumbs, attachments, and every context except `os`, `runtime`, `device` and `trace`. Error messages and panic values are sent as they are, so a handler that puts an ID or a name into an error message sends it too.
