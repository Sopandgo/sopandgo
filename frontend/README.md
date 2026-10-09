# sopandgo frontend

SvelteKit 2 + Svelte 5 application (Node adapter in production). UI uses Tailwind CSS v4 and DaisyUI.

## Scripts

| Command | Purpose |
|--------|---------|
| `npm run dev` | Local dev server |
| `npm run build` | Production build |
| `npm run preview` | Preview production build |
| `npm run check` | `svelte-check` (requires env such as `BACKEND_URL` where used) |
| `npm test` | Vitest |

## Layout: creating a new SOP version

Route: **`/sops/[sop_id]/new`** (`src/routes/(app)/sops/[sop_id]/new/`).

The page uses a responsive grid:

- **Main column:** draft editor (`SopVersionEditor.svelte`).
- **Sidebar:** `SopEditorAssetSidebar.svelte` — Word import card, then assets list and upload.

### Shared draft content

Draft markdown is held in **`+page.svelte`** as `documentContent` and passed with **`bind:content`** to both:

- **`SopVersionEditor`** — textarea, publish form, live preview.
- **`SopEditorAssetSidebar`** — `.docx` import updates the same string so the editor and POST body stay in sync.

### Edit vs preview

The editor uses DaisyUI **`tabs tabs-boxed`** for **Edit** and **Preview**. Only one pane is visible at a time so the draft uses the **full width** of the editor card (no cramped side-by-side preview inside the 2/3 grid column).

- **Preview** reuses **`MarkdownRenderer.svelte`** — the same markdown parsing, `assets/…` image resolution, and HTML sanitization as the published version page (`/sops/.../v/...`).
- Preview text is **debounced** (200 ms) from the textarea; when the server returns `form.content` after a failed publish, preview updates immediately.

### Word import

**Import from Word** lives in the **sidebar**, above **Assets**. Import is only allowed when the draft is **empty** (same rule as before). Errors from import appear in that card.

### Assets

Sidebar lists SOP assets and **Upload Asset** (separate POST `?/upload`). Markdown in the editor can reference files with `![alt](assets/filename.png)`; `MarkdownRenderer` resolves those against the asset list from the page load.

## Notable paths

| Path | Role |
|------|------|
| `src/lib/sdk/` | Typed API client used from `hooks` / `+page.server` |
| `src/lib/components/editor/` | New-version editor and asset sidebar |
| `src/lib/components/MarkdownRenderer.svelte` | Markdown → HTML for SOP bodies (viewer + editor preview) |
| `src/lib/security/sanitizeSopHtml.ts` | DOMPurify profile for `{@html}` SOP output |
| `src/lib/components/Card.svelte` | Every card surface; `variant` (`raised`, `flat`, `inset`, `subtle`) sets background and shadow, `class` only adds layout |
| `src/lib/components/Alert.svelte` | Every alert; `message` for text or children for markup, plus `soft`, `compact`, and `role` |
| `src/lib/components/ListRow.svelte` | Row in a `<ul class="list">`; with `href` the title is a stretched link and `trailing` actions stay clickable |
| `src/lib/components/Combobox.svelte` | Accessible searchable select (WAI-ARIA combobox), used by the audit-log filters |

Use these instead of writing `card` or `alert` classes by hand. All components use `<script lang="ts">`.

## Icons

The Gopher image (`Icon_512.webp`) is based on the Go Gopher created by Renee French, licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
