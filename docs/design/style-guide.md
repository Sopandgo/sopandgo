# SOP and GO style guide

The rules for the frontend's look and language. Token values live in [`tokens.json`](tokens.json) next to this file; token names match the daisyUI theme variables (`base-100` is `--color-base-100`, and so on). The theme itself is in `frontend/src/routes/layout.css`.

SOP and GO is a self-hosted SOP system for labs and small research teams. People open it to find the procedure they need, sign that they have read it, and prove later that they did. The interface should feel like a well-kept lab notebook: calm, flat, precise, and trustworthy enough to show an auditor.

## Principles

- **Calm over cheerful.** One accent (`primary`, the logo's ink), warm neutrals from the logo, no gradients, no decorative shadows. The content is the procedure, not the chrome.
- **Colour means something.** A colour other than `base-content`, `base-content-muted` or `primary` always signals a state: success, warning, error or info. If nothing is wrong or done, it is neutral.
- **Never by colour alone.** Every status carries a word, and integrity states carry an icon too. Success and error must read correctly in greyscale and for colour-blind users.
- **Honest states.** Show exactly what is known: "Hash mismatch" and "Couldn't check" are different states and never share a badge.
- **Flat, bordered, quiet.** Surfaces separate by a `base-100` card on the `base-200` page with a `base-300` hairline. The only shadow is `shadow-overlay`, for things that float.

## Voice

- Sentence case everywhere: headings, buttons, badges, menu items ("Release candidate", "Verify file", not "Verify File").
- Address the reader as "you": "You are all caught up." Name the action and its object: "Sign the latest published version of each SOP below."
- Plain and specific. "Out of date — new version published" beats "Attention needed!".
- One exclamation mark is allowed: the dashboard greeting ("Good morning, {name}!"). Nowhere else.
- No emoji in the interface. Replace the party-popper icon on the dashboard with a check-circle in `success`.
- Status words come from Paraglide messages; badges never uppercase them, so German and CJK translations keep their natural form.
- Labels never use `uppercase tracking-widest`. Group headings inside cards are the `label` style in `base-content-muted`.

## Colour

- Page background is `base-200`. Cards, the navbar, the sidebar, menus and inputs are `base-100`.
- Body text is `base-content`. Metadata, timestamps and help text are `base-content-muted`. Do not dim text with `opacity-40/50/60`: those fail contrast. Use `text-base-content/70` at the least.
- `primary` is the logo's ink (the same value as `base-content` in each theme), so the blue mascot is the only hue in the UI. It is for the main action on a page, the active nav item, the focus ring and selection (`primary-soft` behind it).
- Colour no longer marks links: links are always underlined (`underline`, including in `prose`), in `base-content`. Hover may thicken the underline; it never removes it.
- Because the primary button is now dark ink on paper (light on ink in dark), it stands out only by being the sole filled button. Keep at most one filled `btn-primary` per view; this rule matters more than before. Other actions are `btn-ghost` or outline.
- Selected and active states use `primary-soft` plus a non-colour cue (a 2px `primary` marker, bold label or check icon), not tint alone.
- `secondary` is a near-neutral; use it rarely. `accent` (sand) is decorative only.
- Status colours appear as soft badges and alerts: `success` text on `success-soft`, and so on. Solid status fills (`btn-error`) are only for destructive confirmations.
- `base-300` is for hairlines, never a text ground.
- `brand-sky`, `brand-sand` and `brand-deep` are the logo's colours for the landing page, the cover and PDF exports, not for UI state. `info` is the only blue left in UI state, and only for informational alerts.

### Status mapping

| Meaning | Treatment |
| --- | --- |
| SOP draft | neutral outline badge: `base-content-muted` text, `base-300` border |
| Release candidate | `warning` on `warning-soft` |
| Published | `success` on `success-soft` |
| Rejected | `error` on `error-soft` |
| Superseded | text only in `base-content-muted`, no badge fill |
| Role: admin, approver, auditor | neutral outline badge; no red for admin |
| Integrity verified | `success` soft badge, shield-check icon, "Verified" |
| Hash mismatch | `error` soft badge, shield-x icon, "Hash mismatch" |
| File missing | `warning` soft badge, file-x icon, "File missing" |
| Couldn't check (network, session, server) | neutral outline badge, "Couldn't check", plus a retry button |
| Signature new / out of date | row on `base-100` / row on `warning-soft`, each with its word |

## Type

- One sans family, the system stack (`sans`), so the self-hosted app ships no font files and renders CJK with the OS's own faces. `mono` is for IDs, versions and hashes only.
- Styles: `page-title` (one per page), `section-title` (card headings), `body` (lead text and SOP prose), `body-sm` (the default UI size), `label`, `meta`, `mono-id`.
- `meta` (12px) is the floor: no `text-[10px]`.
- IDs and hashes use `mono-id`, normal case, normal tracking. Truncate hashes to 8 characters and put the full value in a `title`.
- Rendered SOP content uses Tailwind Typography (`prose`) at `body` size with `prose-headings:font-semibold`; links underlined in `base-content`.

## Space, radius, borders, depth

- Spacing is the Tailwind 4px scale; the common steps are `space-2` (button gaps), `space-3`/`space-4` (list rows), `space-6` (card padding and gaps between sections), `space-8` (page gutter on desktop).
- Content column max width `content-max`, centred.
- Radii are small: `radius-selector` 4px, `radius-field` 6px, `radius-box` 8px. Only avatars are round. No `btn-circle` except icon-only toolbar buttons.
- Every border is `border-width` 1px. Cards: `border border-base-300`, no shadow. Lists inside cards separate rows with `border-b border-base-300`.
- `shadow-overlay` for dropdowns, the combobox list and modals only. Remove `shadow-md`, `shadow-lg`, `shadow-2xl` and `shadow-inner` from cards and sections.

## States and motion

- Focus: every interactive element shows `focus-ring` — a 2px solid outline with a 2px offset. Never remove outlines without this replacement.
- Hover on list rows: background `base-200`. Links are always underlined; hover thickens the underline.
- Disabled: daisyUI's default (reduced opacity, no pointer). Keep the label legible.
- Loading: a 16px spinner inside the button that triggered it, label kept ("Checking…"). No full-page spinners for actions that take under a second.
- Motion: colour and background transitions at 150ms. No bounces, no scale effects, nothing that animates on page load.

## Iconography

- Lucide (`lucide-svelte`), stroke width 2 (the default). Do not use `strokeWidth={3}`.
- 16px inline with text, in menus and in badges; 20px as the leading icon in list rows.
- Icons sit beside a word. An icon-only button needs an `aria-label` and a `title`.
- Icons take the colour of their text. Status icons use the status token.

## Layout

- Navbar: `base-100`, a `base-300` bottom border, no shadow. The logo sits at its own size, not inside a circular button.
- Pages: page header (`page-title` plus one line of `body-sm` in `base-content-muted`, actions on the right), then cards stacked with `space-6` gaps.
- Lists, not tables, for SOPs, versions, assets and signatures: a `ListRow` per item inside one bordered card.
- Phones first: rows stack, secondary metadata wraps under the title, badges hide only if their meaning is repeated elsewhere.

## Components

### Card

The bordered `base-100` section that holds every block of content on a page; always rendered through `$lib/components/Card.svelte`.

**Markup**: `card bg-base-100 border border-base-300`, no shadow. Note that daisyUI's own `card-border` draws in `base-200`, which disappears on the `base-200` page, so the component sets the border itself.

**Anatomy**
- Optional header: `section-title` heading plus an optional one-line description in `base-content-muted`, actions on the right, a `base-300` bottom border.
- Body: `card-body` with `space-6` padding (`space-4` on phones). A list inside a card goes edge to edge with no body padding.

**Consumer provides**: `title` (optional), `description` (optional), an `actions` snippet (optional) and `children`.

**Collapsible**: when a page lists sections that are configured once and then left alone (the channels on Settings → Integrations, the SMTP and Resend transports on Settings → Email, S3, validate and apply on Settings → Backup), render them through `$lib/components/CollapsibleCard.svelte`. Its header row is a disclosure: a 20px chevron that turns when open, the `section-title` as a `<button aria-expanded aria-controls>` stretched over the row, and a `meta` line. The section's state stays visible on the collapsed row, as the control itself when it can be changed there (the channel's on/off toggle, raised above the stretched button) or as a badge when it cannot ("Not configured"). Don't use `<details>` for this: a toggle inside `<summary>` is a control nested in a button. Collapsed by default; open the section by itself when a form inside it returns a result, so the message is never hidden. The row hovers to `base-200` and keeps the focus ring inset.

**Don't**
- Don't write `class="card …"` by hand in pages. Today 16 places do; route them through `Card`.
- Don't nest cards. Use a divider or a group heading inside one card.
- Don't add `shadow-*` or `rounded-2xl`; the radius is `radius-box`.

### ListRow

One linked item in a list inside a `Card` — an SOP, a version, an asset, a signature — with a leading icon, a title, a meta line and trailing status or actions. A new shared component replacing the duplicated rows on the dashboard and in `ListSops`, `ListSopVersions` and `ListAssociatedAssets`.

**Anatomy**
- Leading: a 40px `radius-field` tile on `base-200` with a 20px icon in `base-content-muted` (file-text for SOPs, paperclip for assets).
- Main: title in `label` (truncates on one line), meta line in `meta` / `base-content-muted`; IDs and versions in `mono-id`.
- Trailing: a `StatusBadge` or `IntegrityStatus`, small actions (`btn-sm`), and a chevron when the whole row is a link.
- Rows are separated by `base-300` hairlines; the last row has none.

**Variants**: default (`base-100`); `attention` for rows that need the reader's action, such as "Out of date — new version published": background `warning-soft`, leading icon in `warning`, and the reason in words in the meta line.

**Group heading**: above a group of rows, `label` in `base-content-muted`, sentence case, on `base-100` with a bottom hairline. Not uppercase-tracked.

**Consumer provides**: `href` (optional; the whole row becomes a link), `icon`, `title`, `meta` (string or snippet), a `trailing` snippet, and `attention` (boolean).

**Interaction**: hover `base-200`; focus ring inset (`outline-offset: -2px`) so it is not clipped by the card.

### Button

Actions, built on daisyUI `btn`, with at most one filled primary button per view.

**Variants**
- `btn btn-primary` — the one main action of the page or dialog: "Sign", "Publish version", "Save".
- `btn` (default, bordered on `base-100`) — secondary actions: "Cancel", "Compare versions".
- `btn btn-ghost` — tertiary actions in toolbars and list rows: "Download", "Verify file".
- `btn btn-error` — only to confirm a destructive action inside a confirmation dialog ("Revoke all sessions"). The button that opens the dialog is a default `btn` with `text-error`.

**Sizes**: default (`field-height`, 40px) for forms and page headers; `btn-sm` (`field-height-sm`, 32px) inside list rows and card headers.

**Consumer provides**: a sentence-case verb label ("Verify file", not "Verify File"); optionally a 16px Lucide icon before the label. Icon-only buttons (`btn-square btn-ghost btn-sm`) need `aria-label` and `title`.

**Loading**: keep the label, swap the icon for `loading loading-spinner loading-xs`, and disable the button ("Checking…").

**Don't**
- Don't use `btn-circle` for anything but icon-only toolbar buttons.
- Don't put the logo inside a button.
- Don't use `btn-secondary` or `btn-accent`; they add colour without meaning.

### StatusBadge

The SOP version status (`SopVersionStatusBadge`), as a soft daisyUI badge whose colour follows the meaning and whose word always carries it.

| Status | Classes | Word (Paraglide) |
| --- | --- | --- |
| `draft` | `badge badge-outline` | `status_draft` |
| `rc` | `badge badge-soft badge-warning` | `status_rc` |
| `published` | `badge badge-soft badge-success` | `status_published` |
| `rejected` | `badge badge-soft badge-error` | `status_rejected` |
| `superseded` | plain `text-xs text-base-content/70`, no badge | `status_superseded` |
| unknown | `badge badge-outline` | `status_unclear` |

**Consumer provides**: `status` (one of the above). Nothing else; the component owns the mapping, so no other file picks a badge colour for a status.

**Roles** (admin, approver, auditor, reader) use the same neutral `badge badge-outline`. The role is information, not a warning: no red for admin.

**Don't**
- Don't fill badges solid (`badge-primary`, `badge-secondary`).
- Don't uppercase the word or add tracking; translations keep their natural case.
- Don't hide the badge on phones unless the status is stated elsewhere in the row.

### IntegrityStatus

The result of a file or version hash check (`IntegrityButtonAssets`), showing exactly what is known: verified, hash mismatch, file missing, or couldn't check.

| State | When | Treatment |
| --- | --- | --- |
| idle | before a check | `btn btn-ghost btn-sm` with shield icon, "Verify file" |
| checking | request in flight | same button, disabled, spinner, "Checking…" |
| verified | `hash_valid: true` | `badge badge-soft badge-success`, shield-check, "Verified" |
| hash mismatch | `hash_valid: false` | `badge badge-soft badge-error`, shield-x, "Hash mismatch" |
| file missing | backend 404 | `badge badge-soft badge-warning`, file-x, "File missing" |
| couldn't check | network error, 401/403, 500 | `badge badge-outline`, "Couldn't check", plus a ghost "Retry" button |

**Consumer provides**: `sopId` and `assetId` (or `versionId`). The component calls a SvelteKit form action, which calls the SDK with the real `sopId`, and maps the response status to one of the states above.

**Size**: `size="small"` drops the word but keeps the icon and moves the word into `title` and `aria-label`; use it only in dense rows where the column header names the check.

**Don't**
- Never show "Corrupt" for anything but a confirmed hash mismatch. A failed request is not evidence of corruption.
- Don't use colour alone: every state has its icon and word.

### Alert

An inline message about the page or a form, rendered through `$lib/components/Alert.svelte` as `alert alert-soft alert-<status>`.

| Type | Use for | Icon |
| --- | --- | --- |
| `info` | neutral guidance ("Signatures are recorded with your account and the time.") | info |
| `success` | a completed action that changed something lasting ("Version 3 published.") | check-circle |
| `warning` | something the reader should act on soon ("This SOP has a newer version.") | triangle-alert |
| `error` | an action failed or data is invalid ("The file could not be uploaded.") | circle-x |

**Consumer provides**: `type`, a short `title` (optional) and the message. Messages come from Paraglide; form errors from `actionErrorMessages`.

**Placement**: above the content it refers to, full width of its card or form. Not as a toast for routine saves; SvelteKit's form state is enough.

**Don't**
- Don't write `class="alert …"` by hand (14 places do today).
- Don't use solid `alert-error` fills or stack more than two alerts.
- Don't use an alert for empty states; use plain `body-sm` text in the card.

### Avatar

A person's initials in a neutral circle (`$lib/components/Avatar.svelte`), the same for every role.

**Markup**: `avatar avatar-placeholder` with an inner `rounded-full bg-base-200 text-base-content border border-base-300`. Sizes `sm` 32px (lists, signatures), `md` 40px (navbar), `lg` 48px (profile).

**Consumer provides**: `displayName` and `size`. The `color` prop goes away: role is shown by a `StatusBadge`-style outline badge where it matters (the navbar, user lists), never by avatar colour.

**Don't**
- Don't colour avatars red for admins or by any other attribute.
- Don't put the avatar inside `btn-circle`; the dropdown trigger is a plain `button` with the avatar and a visible focus ring.

### TextField

A labelled input, select or textarea: daisyUI `input` / `select` / `textarea` with a visible label above and optional help or error text below.

**Anatomy**: `label` style label (sentence case), the field at `field-height` with a `field-border` outline (the 3:1 override in the theme), then help text in `meta` / `base-content-muted`, or the error in `meta` / `error` with `input-error` on the field.

**States**: focus shows `focus-ring`; error adds `input-error` and `aria-invalid="true"` and links the message with `aria-describedby`; disabled uses daisyUI's default.

**Consumer provides**: `label`, `name`, the field type, `help` (optional), `error` (optional, from the form action's result).

**Don't**
- Don't use placeholders as labels.
- Don't use `input-bordered` or colour variants (`input-primary`) for decoration; the default already has a visible border.

### Toggle and checkbox

A toggle switches something on or off (a notification channel, a user account); a checkbox picks items from a set (the events a channel sends).

**Toggle**: `toggle toggle-success`. On is a state, so it takes `success`; off stays neutral. A word sits next to the toggle ("On" / "Off", "active" / "disabled"), so the state never rests on colour alone.

**Checkbox**: `checkbox checkbox-sm checkbox-primary`: ink fill with a paper check, the selection colour. The plain `checkbox` fills in the 55% `field-border` grey and its check barely shows. Several options go in a `fieldset` with a `label`-style `legend`, in a one-column grid on phones and two columns from `sm`.

**Labels**: wrap the input and its words in a plain `<label class="flex items-center gap-3 text-sm">`. daisyUI 5's `.label` keeps text on one line and dims it below `text-base-content/70`, and `form-control` / `label-text` no longer exist in daisyUI 5.

**Don't**
- Don't add `@tailwindcss/forms`: its base styles draw a second checkmark into daisyUI checkboxes and toggles.
- Don't use `toggle-primary` or `toggle-secondary`; neither says "on".

### ConfirmDialog

The modal that confirms a destructive action, such as applying a backup (`$lib/components/ConfirmDialog.svelte`).

**Markup**: a native `<dialog class="modal">` opened with `showModal()`, so focus moves into it and Escape closes it. The box is `modal-box` on `base-100` with a `base-300` border, `radius-box` and `shadow-overlay`.

**Anatomy**: `section-title` heading (a question: "Apply this backup?"), the consequence as a `warning` `Alert`, any typed confirmation, then Cancel (default `btn`) and the confirm button (`btn btn-error`) on the right.

**Behaviour**: the button that opens it is a default `btn` with `text-error`, and checks the form first (`reportValidity()`). Put the dialog inside the `<form>` it confirms, so its fields are submitted and the confirm button submits. While the action runs, the confirm button shows its spinner and Cancel and Escape do nothing. The dialog closes when the result arrives; the result shows as an `Alert` in the card.

**Consumer provides**: `title`, `open` (bindable), `busy`, the body as `children` and the confirm button as the `confirm` snippet.

**Don't**: put `btn-error` anywhere but inside this dialog, or use it for actions that are easy to undo.

### Combobox

A searchable single-select field for long lists — the audit-log filters for event type, actor and SOP, and later the tag picker — built on Bits UI's headless Combobox and styled with this theme.

**Why Bits UI**: it supplies `role="combobox"`, `aria-expanded`, `aria-activedescendant`, arrow-key and Escape handling, and closing without the current `setTimeout` blur workaround. daisyUI stays responsible for the look.

**Anatomy**: a `TextField`-style label and input (with a chevrons-up-down or clear button at the right), and a listbox below on `base-100` with a `base-300` border, `radius-box` and `shadow-overlay`. Options are `body-sm` rows with `radius-field`; the keyboard-active option shows `focus-ring` inset, the selected option `primary-soft`.

**Consumer provides**: `label`, `items` (`{ value, label, hint? }`, e.g. a user's display name with their role as hint), `value` (bindable), `placeholder`, and `name` for the form.

**Behaviour**: filters as you type; Enter selects; Escape closes, then clears; an empty result shows "No matches" in `base-content-muted`. The selected value's label stays in the input; the id goes into the URL filter.

**Don't**: copy-paste the three current audit-log implementations; replace all three with this one component.

## Implementing in the app (daisyUI 5)

Replace both `@plugin "daisyui"` lines in `frontend/src/routes/layout.css` with:

```css
@plugin "daisyui" { themes: false; }

@plugin "daisyui/theme" {
  name: "sop-light";
  default: true;
  color-scheme: light;
  --color-base-100: #ffffff;
  --color-base-200: #f8f7f2;
  --color-base-300: #d9d5cc;
  --color-base-content: #2c2a29;
  --color-primary: #2c2a29;
  --color-primary-content: #f8f7f2;
  --color-secondary: #5c5853;
  --color-secondary-content: #ffffff;
  --color-accent: #e1cbab;
  --color-accent-content: #2c2a29;
  --color-neutral: #2c2a29;
  --color-neutral-content: #f8f7f2;
  --color-info: #2a5f8f;
  --color-info-content: #ffffff;
  --color-success: #2e6b3c;
  --color-success-content: #ffffff;
  --color-warning: #7f5300;
  --color-warning-content: #ffffff;
  --color-error: #b0281c;
  --color-error-content: #ffffff;
  --radius-selector: 0.25rem;
  --radius-field: 0.375rem;
  --radius-box: 0.5rem;
  --size-selector: 0.25rem;
  --size-field: 0.25rem;
  --border: 1px;
  --depth: 0;
  --noise: 0;
}

@plugin "daisyui/theme" {
  name: "sop-dark";
  prefersdark: true;
  color-scheme: dark;
  --color-base-100: #2c2a29;
  --color-base-200: #222120;
  --color-base-300: #45423e;
  --color-base-content: #f2f1f0;
  --color-primary: #f2f1f0;
  --color-primary-content: #2c2a29;
  --color-secondary: #c9c5be;
  --color-secondary-content: #2c2a29;
  --color-accent: #e1cbab;
  --color-accent-content: #2c2a29;
  --color-neutral: #3d3a37;
  --color-neutral-content: #f2f1f0;
  --color-info: #9cc6e6;
  --color-info-content: #222120;
  --color-success: #93d1a2;
  --color-success-content: #222120;
  --color-warning: #ecc16f;
  --color-warning-content: #222120;
  --color-error: #f2a097;
  --color-error-content: #222120;
  --radius-selector: 0.25rem;
  --radius-field: 0.375rem;
  --radius-box: 0.5rem;
  --size-selector: 0.25rem;
  --size-field: 0.25rem;
  --border: 1px;
  --depth: 0;
  --noise: 0;
}

/* Field borders at 3:1 (daisyUI's default is 20%). Declared in daisyUI's
   base sublayer so input-error, select-primary etc. still win. */
@layer utilities {
  .input, .select, .textarea, .checkbox {
    @layer daisyui.l1.l2.l3 {
      --input-color: color-mix(in oklab, var(--color-base-content) 55%, #0000);
    }
  }
}

/* One focus ring everywhere */
:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
```

In `frontend/src/lib/theme.ts`, `daisyTheme()` returns `'sop-light'` and `'sop-dark'` instead of `'corporate'` and `'business'`. The soft tints (`primary-soft`, `success-soft` …) need no tokens of their own in the app: daisyUI's `badge-soft` and `alert-soft` mix them from the status colours, and Tailwind's `bg-primary/10` covers selections.

### Class swaps

| Today | Becomes |
| --- | --- |
| `card card-border bg-base-100 shadow-md` | `card bg-base-100 border border-base-300` (inside `Card`) |
| `shadow-lg`, `shadow-2xl`, `shadow-inner` on sections | removed |
| `badge badge-primary` (published) | `badge badge-soft badge-success` |
| `badge badge-secondary` (draft) | `badge badge-outline` |
| `badge badge-warning` (RC) | `badge badge-soft badge-warning` |
| `badge badge-error` (rejected) | `badge badge-soft badge-error` |
| `badge badge-lg badge-dash badge-error` (admin) | `badge badge-outline` |
| `Avatar color="error"` / `"secondary"` | `Avatar` neutral: `bg-base-200 text-base-content border border-base-300` |
| `text-[10px] font-mono uppercase tracking-tighter opacity-40` | `font-mono text-xs text-base-content/70` |
| `text-xs font-bold uppercase tracking-widest opacity-60` | `text-sm font-medium text-base-content/70` |
| `PartyPopperIcon text-success` | `CircleCheckIcon text-success` |
| logo inside `btn btn-ghost btn-circle avatar` | plain `<img>` at 32px |
| `LogOutIcon strokeWidth={3}` | default stroke |

## Do and don't

- Do put every card through `Card` and every alert through `Alert`, so this theme changes in one place.
- Do give each status a word. Don't rely on the badge colour.
- Do keep one filled primary button per view. Don't make every action a `btn-primary`.
- Don't use red for roles, avatars or anything that isn't an error.
- Don't add shadows to things that sit in the page.
- Don't introduce new colours in components; reach for a token, or ask whether the state deserves one.
