# 4. Error reporting is configured by admins, and the DSN is public

Status: accepted (2026-11-20)

## Context

Labs that run SOPandGO may want to know when it fails, in a Sentry or GlitchTip they run or pay for. Sending errors out of the instance is a data-protection decision for the lab: stack traces and error messages can contain identifiers. Other outbound destinations (Slack, Gotify, webhooks, S3, the TSA) are set by an admin in the UI and audited.

A Sentry DSN only allows sending events to one project. Sentry documents it as safe to make public, and browser reporting has to put it into every page.

## Decision

- Error reporting is configured under **Settings → Error reporting**, by admins only, and is off by default. There are no environment variables for it. Every change and every test event is audited; the audit log records the DSN's host and project, not the DSN.
- The DSN is stored in plain text, not sealed with `SECRET_ENCRYPTION_KEY`. While reporting is on, `GET /api/error-reporting/config` returns it to anyone, because the SvelteKit server reads it there and browsers need it when browser reporting is on. The admin page says so.
- Events carry no personal data by design: no request, user, IP, headers, cookies, bodies, query strings or breadcrumbs. Error messages and stack traces are sent as they are.
- Browser reporting is a separate switch, off by default.
- A backup restore keeps the running instance's settings, like the S3 and anchoring settings, because the DSN names this instance's project.

## Consequences

- The public demo has no error reporting: its admin area is closed and its database is wiped at every start.
- Anyone who can reach an instance with reporting on can send events to its project. The cost is noise in that project, which the project's own rate limits bound.
- Turning reporting on needs no restart, and does not need `SECRET_ENCRYPTION_KEY`.
