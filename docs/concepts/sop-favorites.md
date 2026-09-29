# SOP favorites

## What it is

**Favorites** are per-user bookmarks for **SOP containers** (the logical SOP), not for individual versions. Marking an SOP as favorite applies regardless of which version you are viewing.

## Privacy

- Each favorite row is tied to **`user_id` + `sop_id`** in SQLite (`sop_favorites`).
- Other users cannot see your favorites; list and detail APIs only expose `is_favorite` for the **authenticated** user.

## Where it appears in the UI

- **Home dashboard** (`/dashboard`): quick grid of favorite SOPs (with unfavorite), plus a link to the full favorites-only list. See `docs/concepts/home-dashboard.md`.
- **SOP list** (`/sops`): star to favorite or unfavorite; optional filters **Favorites only** and **Favorites first** (URL query params `favorites_only=true`, `favorites_first=true`).
- **SOP detail** (`/sops/{id}`): add/remove favorite for that SOP.
- **Version view** (`/sops/{id}/v/latest` or `/sops/{id}/v/{versionId}`): same SOP-level favorite control in the header card.

## Data and backups

- Favorites live in **`app.db`** (table `sop_favorites`). Full-directory and admin **Backup** exports include them automatically with the database snapshot.
- Deleting a user or SOP removes related favorite rows (cascade per schema).

## Audit

The audit log may record **`sop_favorite_added`** and **`sop_favorite_removed`** when a favorite row is actually inserted or removed (idempotent repeats do not duplicate rows).

Favorites are exposed on SOP list/detail handlers and `POST`/`DELETE .../favorite` in `backend/internal/api/api.go`. See `docs/dev/api-reference.md` for how the HTTP surface is documented.
