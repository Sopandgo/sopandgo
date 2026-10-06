# Releasing

sopandgo uses [semantic versioning](https://semver.org/) from **1.0.0** onward. The product version is an annotated git tag such as `v1.0.1`. Root and frontend `package.json` versions are not the product version.

## A merge is not a release

After a feature or fix:

1. Do the work on a branch.
2. Merge into `main` when checks are green.
3. Stop.

`main` is the latest code. Labs do not receive that merge. Several changes can land before any of them ship. Pushing to `main` runs tests and builds the container image. It does not publish a tag or an image.

## A tag is what labs can run

Do this only when `main` is ready for other people. One tag can include many merged changes.

1. On `main`, add a heading to [CHANGELOG.md](../../CHANGELOG.md). The heading is the version **without** a `v`, for example `## 1.0.1`. Under it, describe what changed.
2. Checks on `main` are green.
3. Tag that commit and push the tag. Tag `main` after the merge, not the pull request branch.

```bash
git tag -a v1.0.1 -m "v1.0.1"
git push origin v1.0.1
```

4. The [Release workflow](../../.github/workflows/version-and-release.yml) reads that changelog section and uses it as the GitHub Release text. It does not write a second set of notes from commit titles. It then builds `linux/amd64` and `linux/arm64` images and pushes them to `ghcr.io/sopandgo/sopandgo`. The image is stamped with `APP_VERSION` from the tag (`1.0.1`), which the footer and backup manifests show.
5. A lab gets it the next time they export a backup and run `docker compose pull` and `docker compose up -d`. They do not edit a version.

For a normal 1.x release, do not edit `.env`, `package.json`, or the image line in `docker-compose.yml`.

### What a stable tag publishes

A tag like `v1.2.3` (exactly three numbers) publishes:

| Image tag | Who it is for |
| --- | --- |
| `1.2.3` | That exact release. Use it to roll back. It never moves. |
| `1.2` | Latest release within 1.2.x. |
| `1` | The lab channel. It moves to each new 1.x release and stops when 2.0.0 exists. |
| `latest` | Newest stable release, so `docker pull ghcr.io/sopandgo/sopandgo` works. Lab installs do not use this name. A future 2.0.0 moves `latest` and does not move `1`. |

`docker-compose.yml` is pinned to `ghcr.io/sopandgo/sopandgo:1`.

### Prerelease tags

A tag like `v1.2.0-rc.1` must have a matching `## 1.2.0-rc.1` changelog heading. The workflow marks the GitHub Release as a prerelease and pushes only `ghcr.io/sopandgo/sopandgo:1.2.0-rc.1`. It does not move `1`, `1.2`, or `latest`.

### Publishing a tag that already exists

The workflow can be run by hand on an existing tag (Actions → Release → Run workflow) without moving or deleting that tag. Use this to publish an image for a tag that was created before this workflow existed.

After the first successful publish, open GitHub Packages and confirm the `sopandgo` container is **public**. The workflow cannot change a private package to public by itself.

## What not to do

- Do not tag every merge. A tag means labs may run this now.
- Do not move or delete a `v*` tag. If the build was wrong, merge the fix and ship the next patch (`v1.0.2`).
- Do not invent a version in docs without a matching git tag.
- Do not auto-number from commit messages. Tags are the source of truth.
- Do not publish this project to npm.
- Source builds (`docker-compose.dev.yml`) and local `npm run dev` are not releases. The footer may say `dev` or a `git describe` string.
