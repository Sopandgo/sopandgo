# branding

The organization's name, logo, and primary colour, which replace the SOPandGO defaults in the web app and in outgoing messages. Admins change them under **Settings → Branding**.

## Storage

One row (`id = 1`) in the `branding` table (migration v29). `NULL` means the default.

| Column | Meaning |
| --- | --- |
| `name` | Organization name, at most 40 characters, one line |
| `logo_png` | Logo, aspect ratio kept, longest edge at most 512 px |
| `icon_png` | 64 × 64 favicon, the logo centred on transparent padding |
| `logo_hash` | SHA-256 of `logo_png`; ETag and cache-buster |
| `primary_light`, `primary_dark` | `#rrggbb` per theme |

The images are BLOBs in `app.db`, so backups and restores include them without extra files.

## Rules

- **Logo:** PNG, JPEG, or WebP, at most `MaxUploadBytes` (2 MB) and 4096 px per side, checked before the pixels are decoded (`backend/internal/imageutil`). The upload is decoded and re-encoded as PNG, so only pixels survive. SVG is refused.
- **Colour:** must reach `MinContrast` (4.5:1) against the theme's `base-100` (`LightBase100`, `DarkBase100`). The content colour (text on a primary button) is that `base-100`, which passes by symmetry.
- **Name:** trimmed; control characters are refused, so it cannot inject mail headers. Any spelling of the product name, split or in any case, counts as no custom name, so the product always reads "SOPandGO".

## Service

| Method | Effect | Audit event |
| --- | --- | --- |
| `Get` | Public view (`Branding`) | — |
| `UpdateText` | Name and both colours in one save | `branding_updated` |
| `SetLogo` | Process and store logo and icon | `branding_logo_updated` |
| `RemoveLogo` | Back to the gopher (`ErrNoLogo` if none) | `branding_logo_removed` |
| `Reset` | Back to every default in one step; a no-op when nothing is set | `branding_reset` (with the old values) |
| `Image` | PNG bytes and hash for `logo` or `icon` | — |
| `Load` | Copy the stored name into `i18n` at startup | — |

Every change is audited in the same transaction and then calls `i18n.SetAppName`, which fills `{app_name}` in email and notification texts.

## HTTP

`GET /api/branding` and `GET /api/branding/logo?variant=logo|icon` are public, because the sign-in page shows them. Writes are under `/api/admin/settings/branding` and need the admin scope. See `backend/internal/api/handle_branding.go`.
