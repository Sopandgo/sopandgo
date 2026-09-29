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
