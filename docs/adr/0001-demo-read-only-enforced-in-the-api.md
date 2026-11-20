# 1. The demo's read-only guarantee is enforced in the Go API

Status: accepted (2026-11-19)

## Context

The public demo (`DEMO_MODE=true`) lets anyone on the internet use SOPandGO without signing in. Every seeded account has a known password, and every instance has `admin`/`admin` until the first sign-in. Caddy exposes the Go API directly under `/sopandgo/*`, past the SvelteKit app. A lock that lives only in the UI, such as hidden buttons or SvelteKit actions that refuse, would stop nobody who calls the API with `curl`.

## Decision

The Go API enforces read-only on its own. A demo guard wraps the whole HTTP router in the API constructor, so `Serve()` and the tests run the same handler. It lets reads through, closes `/api/admin/*` except the two audit log reads, and allows only three writes: opening a demo session, refreshing it and signing out. Everything else gets `403` with the error kind `DemoReadOnly` and the code `demo_read_only`. Password sign-in, the 2FA steps and password reset are among the refused writes, so no token for an admin, editor or approver can be obtained.

The frontend only explains: it shows write controls disabled, with a tooltip, and turns the `demo_read_only` code into a readable message.

## Consequences

- Routes added later are refused in demo mode by default, because the guard sits in front of the router and works from an allowlist. The API tests walk the route table, so a new write route is covered without editing a list.
- Admin reads that write (the integrity scan audits itself) or leak (the backup export) are covered by the same rule, without special cases.
- The demo session endpoint is a password-free sign-in path. It exists only when demo mode is on, maps to two fixed accounts, and a test asserts it is `404` otherwise.
