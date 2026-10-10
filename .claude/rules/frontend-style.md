---
paths:
  - "frontend/src/**/*.{svelte,css,ts}"
---

# Frontend style

Follow `docs/design/style-guide.md`. Read it before any change to markup, classes or the theme. Token values are in `docs/design/tokens.json`.

- Themes are `sop-light` and `sop-dark` (daisyUI 5, defined in `frontend/src/routes/layout.css`). Never add new colours in components; use theme tokens.
- Cards go through `Card`, alerts through `Alert`. No hand-written `card` or `alert` classes in pages.
- Flat surfaces: `bg-base-100 border border-base-300` on the `base-200` page. No shadows except menus, the combobox list and modals.
- At most one `btn-primary` per view. No `btn-secondary`, `btn-accent` or decorative `btn-circle`.
- Statuses through `SopVersionStatusBadge` (`badge-soft` per meaning, words always shown). Roles and avatars stay neutral; red only for errors.
- Integrity checks show four distinct states: verified, hash mismatch, file missing, couldn't check. Never "Corrupt" for a failed request.
- Text never smaller than `text-xs` and never dimmed below `text-base-content/70`. IDs and hashes in `font-mono text-xs`, normal case.
- Sentence case everywhere, no uppercase tracking, no emoji. All copy through Paraglide messages.
- Lucide icons at default stroke, 16px inline, 20px in list rows; icon-only buttons need `aria-label` and `title`.
- Every interactive element keeps the global focus ring.
