---
name: release
description: >-
  Decide whether main is ready for a SOPandGO release, choose the next
  semantic version, and draft the changelog section. Use when the user asks
  about a release, a version tag, CHANGELOG.md, shipping to labs, or cutting
  a v* tag.
---

# Release

`docs/ops/releasing.md` is the source of truth for how a tag is published. Read it, `CHANGELOG.md`, and `.github/workflows/version-and-release.yml` before advising or tagging. `AGENTS.md` already forbids creating, pushing, moving, or deleting a `v*` tag unless the user explicitly asked to cut a release.

## Decide whether main is ready

Compare `main` with the latest `v*` tag. A tag means labs may run this now. Several merges can wait for one tag.

1. Confirm the checkout is `main` and matches `origin/main`.
2. List commits since the latest `v*` tag (`git describe --tags`, `git log <tag>..HEAD`).
3. Note new migrations in `backend/internal/storage/migrations.go` and any operator steps in `docs/ops/deployment.md`.
4. Check that CI succeeded on the commit that would be tagged (`gh run list --branch main`).
5. Look at open pull requests. A development-only dependency bump does not block a release.

Answer with the recommended version, what the batch contains, and whether a matching `## X.Y.Z` heading already exists. Do not edit `CHANGELOG.md` or create a tag while only advising.

## Choose the version

Use semantic versioning. The product version is the git tag. Root and frontend `package.json` versions are not the product version.

- Compatible features since the last tag: the next minor (`1.2.0`). Append-only migrations with a safe default stay on 1.x.
- Fixes only: the next patch (`1.1.2`).
- A break that must stop the lab channel `ghcr.io/sopandgo/sopandgo:1`: a major (`2.0.0`). That channel moves on every 1.x release and stops when 2.0.0 exists.
- A prerelease (`v1.2.0-rc.1`) only when the user asks for one. It does not move `1`, `1.2`, or `latest`.

## Draft the changelog

Add `## X.Y.Z` at the top of `CHANGELOG.md`, without a `v`. The Release workflow uses that section as the GitHub Release text and exits when the heading is missing. Match the existing voice: one opening sentence, then bullets in product language. Write the product name as **SOPandGO**. Add `### Upgrading from …` when migrations or operator steps changed (backup first, what starts automatically, what an admin must set again).

```markdown
## 1.2.0

One sentence on what labs get.

- What changed, in the language of the product.
- Another change.

### Upgrading from 1.1.x

- Migrations run on startup. Take a backup before pulling.
```

## Cut the release

Do this only after the user explicitly asks to cut a release. That request includes the changelog commit, the annotated tag, and pushing the tag.

1. On `main`, commit the changelog heading. Do not tag a pull request branch.
2. Wait until checks on that commit are green.
3. Tag that commit and push the tag:

```bash
git tag -a v1.2.0 -m "v1.2.0"
git push origin v1.2.0
```

4. Confirm the Release workflow publishes the image. Labs receive it the next time they export a backup and run `docker compose pull` and `docker compose up -d`.

Leave `.env`, both `package.json` files, and the `ghcr.io/sopandgo/sopandgo:1` line in `docker-compose.yml` unchanged. If a published build was wrong, merge a fix and ship the next patch. Do not move or delete the tag.
