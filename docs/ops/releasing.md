# Releasing

sopandgo uses [semantic versioning](https://semver.org/) from **1.0.0** onward. Prefer tagged releases for deployments (see the README upgrading notes and [CHANGELOG.md](../../CHANGELOG.md)).

## How versions are cut

Releases are **intentional**. Pushing to `main` runs CI only; it does **not** create a version tag.

1. Land the changes on `main` with CI green.
2. Update [CHANGELOG.md](../../CHANGELOG.md) for the release.
3. Create an annotated tag and push it:

```bash
git tag -a v1.0.1 -m "v1.0.1"
git push origin v1.0.1
```

4. The [Release workflow](../../.github/workflows/version-and-release.yml) creates a GitHub Release for that `v*` tag.

Set `APP_VERSION` in release image builds to the tag **without** the leading `v` (for example `1.0.1`) so the UI footer and backup manifests match the release.

## What not to do

- Do not invent ad-hoc version numbers in docs without a matching git tag.
- Do not rely on auto-bumping from commit messages; tags are the source of truth.
