# Backup Package

Export, validate, and stage-restore of the app data directory, plus optional scheduled S3 uploads.

Operator-facing docs: `docs/ops/backup-and-restore.md`. HTTP surface: `/api/admin/backups/*` in `backend/internal/api/api.go` (see `docs/dev/api-reference.md`).

## Pieces

| File | Role |
| --- | --- |
| **`service.go`** | `ExportZip`, `ValidateImportArchive`, `StageImportApply`, maintenance lock, pending restore under `data/_restore_pending/` |
| **`s3_scheduler.go`** | Optional timed uploads of the same `.zip` format (`BACKUP_S3_*` env vars) |

## Flow (short)

1. **Export** — consistent SQLite snapshot + `sops/` + `manifest.json` into a `.zip`.
2. **Validate** — check archive and schema compatibility without touching live data (`422` on failure via API).
3. **Apply** — requires confirmation string `APPLY BACKUP`; stages under `_restore_pending/` and writes a pre-apply snapshot; swap happens on **next process start**.

Mutating APIs may return `503` while the maintenance lock is held during export/apply.
