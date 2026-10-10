# Demo data

`cmd/seed-demo-data` inserts these SOPs, plus three demo users, on the first start when `app.db` does not exist yet and `SEED_DEMO_DATA` is not `false`. [docs/ops/evaluating.md](../../docs/ops/evaluating.md) describes what an evaluator sees.

## Layout

Each folder under `sops/` is one SOP. The folder name is not used as the SOP ID.

```
sops/<folder>/
  version-1.md     # first line must be "# Title"; the title comes from version-1.md
  version-2.md     # optional further versions, created in file name order
  assets/          # optional images referenced from the Markdown as assets/<file>
  sop.json         # optional, see below

avatars/
  manager.jpg      # profile picture for manager@demo.local
  qa.jpg           # profile picture for qa@demo.local
  researcher.jpg   # profile picture for researcher@demo.local
  ATTRIBUTIONS.md  # origin of the avatar files
```

## sop.json

All fields are optional. Without the file, every version is published and nothing else is set.

```json
{
  "tags": ["Safety"],
  "final_state": "rc",
  "change_summaries": ["Initial release", "Added cycle log"],
  "read_by": { "researcher": [1] },
  "favorited_by": ["researcher"]
}
```

| Field | Meaning |
| --- | --- |
| `tags` | Tags attached to the SOP. A tag is created the first time any SOP uses it. |
| `final_state` | Where the last version stops: `published` (default), `rc`, or `draft`. Earlier versions are always published. |
| `change_summaries` | One change summary per version, in file order. Missing entries default to "Updated procedure". |
| `read_by` | Demo user key to the version numbers that user signed as a reader. A version must be published to be signed. |
| `favorited_by` | Demo user keys that favorite the SOP. |

Demo user keys are `manager` (editor, authors every version), `qa` (approver, publishes every version), and `researcher` (viewer). The bootstrap admin is not given a demo avatar. When `avatars/<key>.jpg` is present, the seeder uploads it through the same profile-picture pipeline as the product (four JPEG sizes under `DATA_DIR/users/`).

The seeder rejects unknown fields and invalid values. `go test ./cmd/seed-demo-data` seeds this folder into a temporary directory, so run it after changing demo data.

## Content

Keep demo SOPs short, mark the title with "(Demo)", and include the disclaimer line used in the existing files. Use original text. Third-party images need an `ATTRIBUTIONS.md` in the SOP's `assets/` folder and a mention in the README's Third-Party Assets section.
