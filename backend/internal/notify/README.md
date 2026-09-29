# Notify Package

Outbound instance-level notifications for Slack Incoming Webhooks, Gotify, and generic HTTP webhooks.

Credentials and event subscriptions live in SQLite (`integration_settings`), encrypted with the same `SMTP_SECRET_ENCRYPTION_KEY` used for mail secrets. There are no per-channel env enable flags — unconfigured or disabled channels simply do not send.

## Channels

| Channel | Delivery |
| --- | --- |
| `slack` | POST JSON `{"text":"..."}` to an Incoming Webhook URL (HTTPS) |
| `gotify` | POST `{title,message,priority}` to `{url}/message?token=...` |
| `webhook` | POST a stable JSON envelope (`event`, `occurred_at`, `title`, `message`, `url`, `sop_id`, …) with optional Bearer token |

## Events

Subscribable types: `sop_published`, `sop_rc`, `sop_rejected`, `backup_s3_failed`, `integrity_check_failed`. Admin test pings use `test` (not stored as a subscription).

## Core pieces

1. **`SettingsStore`** — encrypted CRUD + public DTO (no secrets returned).
2. **`Service.Dispatch`** — fan-out after domain commit; failures are audited (`notification_sent` / `notification_failed`) and never roll back the domain action.
3. **`Service.SendTest`** — pings one channel using stored credentials (even if currently disabled).

Admin HTTP handlers live under `/api/admin/settings/integrations/*` (see `backend/internal/api/api.go`). Operator docs: `docs/ops/deployment.md`. Navigating the HTTP surface: `docs/dev/api-reference.md`.
