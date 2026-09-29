# Contributing

Thanks for helping improve sopandgo. This project targets small labs that need versioned SOPs with acknowledgments — keep changes aligned with that scope (see the README “What this is not” section).

## Development setup

1. Copy `.env.example` → `.env` and set at least `ORIGIN` and `SMTP_SECRET_ENCRYPTION_KEY` for full mail testing.
2. Prefer Docker for an end-to-end stack: `docker compose up -d --build`
3. Or run components separately:
    - Backend: `cd backend && go run cmd/sopandgo/main.go`
    - Frontend: `cd frontend && npm install && npm run dev`

New to the product surface? [docs/ops/evaluating.md](docs/ops/evaluating.md) covers demo users and a short local tour.
## Tests

Run before opening a PR:

```bash
cd backend && go test ./...
cd frontend && npm test
```

## Pull requests

- Keep PRs focused; prefer small, reviewable changes.
- Update docs under `docs/` when behavior or APIs change.
- Do not commit `.env`, `backend/data/`, or real secrets.
- Database migrations are **append-only** (`backend/internal/storage/migrations.go`) — never edit a shipped migration; add a new version instead.

## Scope guidance

In scope: reliability, clarity, self-hosted ops, and the publish → acknowledge → coverage loop.

Out of scope for typical contributions: SSO/LDAP, multi-tenant SaaS, Slack bots (OAuth/interactive apps), LMS/quizzes, or QMS/compliance claims. Outbound Incoming Webhooks / Gotify / generic HTTP push (admin Settings) are in scope when aligned with existing notify patterns.
