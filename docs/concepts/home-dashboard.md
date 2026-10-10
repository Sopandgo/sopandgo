# Home dashboard

The **home dashboard** (`/dashboard`) is the default landing page after a successful login for users in the main app shell. It gives a single place to see what needs attention and to open frequently used SOPs.

## What it shows

1. **Welcome banner** — Time-of-day greeting using the signed-in user’s display name and the browser’s time zone (remembered in a `tz` cookie so the server-rendered page matches), plus a short status line derived from signature data (all caught up vs. how many SOPs still need review).

2. **All SOPs** — The page’s only primary button, in the banner, opens the full SOP library at `/sops` without going through favorites.

3. **Action required** — Lists published SOPs where the user has **not** acknowledged the latest published version, in two groups (same headings, order, and badges as the profile page):
   - **Out of date — new version published:** the user signed an older published version; a newer one exists. These rows are highlighted.
   - **New — not signed yet:** no acknowledgment on any published version of that SOP.

   Each row links to **`/sops/{sop_id}/v/latest`** so the user lands on the current published content and signing flow. With nothing to sign, the card says so instead of listing rows.

4. **What’s new in the lab** — Recent currently published versions (`GET /api/activity/publishes`): title, version, change summary (cut to two lines), who published it, and when. Each row opens **`/sops/{sop_id}/v/latest`**.

5. **Who still needs to sign** — Shown to **admin** and **approver** only. For each published SOP, active users who can reader-sign are split into signed and not yet signed (`GET /api/training/coverage`). SOPs with the most unsigned people come first; each row has a badge with the unsigned count, or “All signed”. SOPs without a published version are left out.

6. **Favorite SOPs** — A list of the user’s starred SOPs (up to 24, from `GET /api/sops?favorites_only=true`). Each row opens the latest version, shows up to three tags (each opens `/sops` filtered by that tag; the rest sit behind “+N”), and has a star to remove the favorite. A link to **`/sops?favorites_only=true`** opens the full filtered list; with more than 24 favorites, the card header says how many are shown. With no favorites, the card explains how to add one and links to `/sops`.

## Data sources

The dashboard loader uses the same APIs as other UI. Each card loads on its own: if one request fails, that card shows an error and the rest of the page still loads. While signature status is unavailable, the banner leaves out its status line instead of claiming you are caught up.

| UI need | API (via SDK) |
| --- | --- |
| Signature groups | `GET /api/auth/me/signature-status` |
| Recent publishes | `GET /api/activity/publishes` |
| Team signature gaps (admin, approver) | `GET /api/training/coverage` |
| Favorites list | `GET /api/sops?favorites_only=true` (`limit=24`) |

Detailed field semantics for signature status are described in `docs/concepts/versioning-and-signatures.md`. Route registration lives in `backend/internal/api/api.go`; see `docs/dev/api-reference.md` for how to navigate the HTTP surface.

## Related pages

- **`/profile`** — Account details and the same signature breakdown in a **Training / Signatures** card. Language, appearance, and password change are on **`/profile/settings`**.
- **`/sops`** — Browse, search, tag filters, and favorite toggles for the whole library.
