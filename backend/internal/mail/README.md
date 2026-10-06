# Mail Package Architecture

This package sends operational email (invites, password resets, publish notices, test mail) and persists encrypted SMTP / Resend settings.

Delivery mode and transport are chosen in the admin UI (stored in SQLite), not via `SMTP_HOST`-style env vars. Secrets are encrypted with `SMTP_SECRET_ENCRYPTION_KEY`.

## Modes and transports

| Setting | Values | Meaning |
| --- | --- | --- |
| `mail_mode` | `smtp`, `manual_links` | Whether the app sends invite/reset mail, or returns one-time links to admins. |
| `mail_transport` | `smtp`, `resend` | Which provider sends when `mail_mode` is `smtp`. |
| `default_locale` | supported BCP 47 tag | Organization language for the test email. Invite, reset, and publish notices use the recipient's `users.locale` instead. |

Subjects and boilerplate come from `backend/internal/i18n`. SOP titles and change summaries are inserted unchanged. An unknown locale or a missing key falls back to English.

Publish notices (`SendSOPPublishedEmail`) run only when mail mode is not `manual_links`. A failed send does not unpublish the version.

## Core Principles

1. **`Service` (`service.go`, `mail_service.go`):** High-level send helpers that use a `Sender` and write audit events via an `Auditor` interface.
2. **Settings store (`smtp_store.go`):** Loads/saves SMTP and Resend credentials (AES-GCM ciphertext in SQLite) and mail mode/transport.
3. **Transports:** `smtp.go` (classic SMTP), `resend_send.go` (Resend HTTP API), `db_sender.go` (resolves the active transport from DB settings).

## File Structure

| File | Purpose |
| --- | --- |
| **`service.go`** | `NewService`, `Service` + `Sender` / `Auditor` interfaces |
| **`mail_service.go`** | `SendUserWelcomeEmail`, `SendPasswordResetEmail`, `SendSOPPublishedEmail`, … |
| **`smtp_store.go`** | Encrypted settings CRUD and mail mode/transport |
| **`smtp.go`** | SMTP `Sender` implementation |
| **`resend_send.go`** | Resend `Sender` implementation |
| **`db_sender.go`** | Sender that picks SMTP or Resend from stored settings |
| **`model.go`** | Shared types |
| **`errors.go`** | Package errors |

## Usage (conceptual)

```go
mailService, err := mail.NewService(sender, auditLogger)
err = mailService.SendUserWelcomeEmail(email, userID, displayName, inviteURL, locale)
err = mailService.SendPasswordResetEmail(email, userID, displayName, resetURL, locale)
err = mailService.SendSOPPublishedEmail(email, name, sopTitle, version, summary, link)
```

Admin API handlers under `/api/admin/settings/*` own persistence of credentials and mode switches; see `backend/internal/api/api.go`, `docs/dev/api-reference.md`, and `docs/ops/deployment.md`.
