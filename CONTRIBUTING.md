# Contributing

Thanks for helping improve sopandgo. This project targets small labs that need versioned SOPs with acknowledgments — keep changes aligned with that scope (see the README “What this is not” section).

## Development setup

1. Copy `.env.example` → `.env` and set at least `ORIGIN` and `SECRET_ENCRYPTION_KEY` for full mail testing.
2. Build the app from this checkout (labs pull the published image instead; see [docs/ops/releasing.md](docs/ops/releasing.md)):

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build
```

That local image is tagged `ghcr.io/sopandgo/sopandgo:1` on your machine. Run `docker compose pull` when you want the published image back.

3. Or run components separately:
    - Backend: `cd backend && go run cmd/sopandgo/main.go`, or `cd backend && air` for live reload. Air runs the demo seeder before each build when `backend/data/app.db` is missing (honors `SEED_DEMO_DATA=false` in `backend/.env`). To re-seed, stop air and delete `backend/data`. Air loads `backend/.env`: copy `backend/.env.example` and set `SECRET_ENCRYPTION_KEY` there to test mail and integrations
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

## Releases

Merging to `main` is not a release. Labs do not receive a change until someone tags `main` after the merge. Do not tag from a pull request branch.

The steps, image names, and rules are in [docs/ops/releasing.md](docs/ops/releasing.md).

## Scope guidance

In scope: reliability, clarity, self-hosted ops, and the publish → acknowledge → coverage loop.

Out of scope for typical contributions: SSO/LDAP, multi-tenant SaaS, Slack bots (OAuth/interactive apps), LMS/quizzes, or QMS/compliance claims. Outbound Incoming Webhooks / Gotify / generic HTTP push (admin Settings) are in scope when aligned with existing notify patterns.
