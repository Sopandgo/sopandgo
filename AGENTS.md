# Agent notes

## Keep the docs in the same change

When behavior, APIs, security guarantees, or operator steps change, update the matching docs in that same change. Do not leave the code and the docs describing different systems.

Check these when they apply:

- `README.md` — what the product does, and the short security summary
- `docs/dev/api-reference.md` — how to find the HTTP surface (code + SDK); not an endpoint encyclopedia
- `docs/dev/architecture.md` and `docs/dev/auth-architecture.md` — how requests, tokens, and roles work
- `docs/concepts/security-model.md` — what the system protects, and what it does not
- `docs/concepts/versioning-and-signatures.md` — lifecycle and acknowledgments
- `docs/ops/deployment.md` — env vars, ports, and how to run it
- `docs/ops/evaluating.md` — local trial with demo data
- `docs/ops/releasing.md` — how version tags and GitHub Releases work
- Package `README.md` files under `backend/internal/` when a package's public behavior changes

`CONTRIBUTING.md` states the same rule for human contributors.

## Releases

Merging to `main` is not a release. Do not create a tag, edit `CHANGELOG.md` for a version, or tell the user that labs received the change unless they explicitly asked to cut a release.

- Do not create, push, move, or delete `v*` tags unless the user explicitly asked to cut a release. When they do, follow `docs/ops/releasing.md`: changelog heading, tag `main`, push the tag. Do not tag the pull request branch.
- Do not bump `package.json` versions as the product version, and do not add commit-message auto-tagging or npm publish.
- A release requires a `## X.Y.Z` section in `CHANGELOG.md` first.
- The lab image tag is `ghcr.io/sopandgo/sopandgo:1` inside `docker-compose.yml`. Do not move that tag to `.env`, to `latest`, or to a patch number.
- Operator steps stay in `docs/ops/releasing.md`. Workflow changes update that file, `CONTRIBUTING.md`, `README.md`, `docs/ops/deployment.md`, and `docs/ops/evaluating.md` in the same change.
