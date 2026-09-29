# Home dashboard

The **home dashboard** (`/dashboard`) is the default landing page after a successful login for users in the main app shell. It gives a single place to see what needs attention and to open frequently used SOPs.

## What it shows

1. **Welcome banner** — Time-of-day greeting using the signed-in user’s display name, plus a short status line derived from signature data (all caught up vs. how many SOPs still need review).

2. **All SOPs** — A primary control opens the full SOP library at `/sops` without going through favorites.

3. **Action required** — Lists published SOPs where the user has **not** acknowledged the latest published version, in two groups (same rules as the profile page):
   - **New — not signed yet:** no acknowledgment on any published version of that SOP.
   - **Out of date — new version published:** the user signed an older published version; a newer one exists.

   Each row links to **`/sops/{sop_id}/v/latest`** so the user lands on the current published content and signing flow.

4. **What’s new in the lab** — Recent currently published versions (`GET /api/activity/publishes`): title, version, change summary, who published it, and when. Each row opens **`/sops/{sop_id}/v/latest`**.

5. **Who still needs to sign** — Shown to **admin** and **approver** only. For each published SOP, active users who can reader-sign are split into signed and not yet signed (`GET /api/training/coverage`).

6. **Favorite SOPs** — A compact grid of the user’s starred SOPs (from `GET /api/sops?favorites_only=true`), with links to latest version and controls to remove a favorite. A link to **`/sops?favorites_only=true`** opens the full filtered list.

## Data sources

The dashboard loader uses the same APIs as other UI:

| UI need | API (via SDK) |
| --- | --- |
| Signature groups | `GET /api/auth/me/signature-status` |
| Recent publishes | `GET /api/activity/publishes` |
| Team signature gaps (admin, approver) | `GET /api/training/coverage` |
| Favorites for the grid | `GET /api/sops?favorites_only=true` (with a reasonable `limit`) |

Detailed field semantics for signature status are described in `docs/concepts/versioning-and-signatures.md`. Route registration lives in `backend/internal/api/api.go`; see `docs/dev/api-reference.md` for how to navigate the HTTP surface.

## Related pages

- **`/profile`** — Full account view; includes the same signature breakdown in a **Training / Signatures** card plus password change.
- **`/sops`** — Browse, search, tag filters, and favorite toggles for the whole library.
- **`/sops/{sop_id}`** — The same team signature list for one SOP, for admin and approver (`GET /api/sops/{sop_id}/training`).
