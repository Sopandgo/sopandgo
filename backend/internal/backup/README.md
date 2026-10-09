# Backup Package

Export, validate, and stage-restore of the app data directory, plus optional scheduled S3 uploads.

Operator-facing docs: `docs/ops/backup-and-restore.md`. HTTP surface: `/api/admin/backups/*` and `/api/admin/settings/backup-s3` in `backend/internal/api/api.go` (see `docs/dev/api-reference.md`).

## Pieces

| File | Role |
| --- | --- |
| **`service.go`** | `ExportZip`, `ValidateImportArchive`, `StageImportApply`, maintenance lock, pending restore under `data/_restore_pending/` |
| **`s3_scheduler.go`** | Optional timed uploads of the same `.zip` format. Always running; `Configure` switches it on, off or to new settings without a restart. `RunNow` for a manual upload, `CheckS3Connection` writes and deletes a probe object |
| **`s3_settings.go`** | `S3SettingsStore`: the admin-UI settings in `backup_s3_settings` (secret sealed with `SECRET_ENCRYPTION_KEY`), and the carry-over that keeps them across a staged restore |

## Flow (short)

1. **Export** — consistent SQLite snapshot + `sops/` + `manifest.json` into a `.zip`.
2. **Validate** — check archive and schema compatibility without touching live data (`422` on failure via API).
3. **Apply** — requires confirmation string `APPLY BACKUP`; stages under `_restore_pending/` and writes a pre-apply snapshot; swap happens on **next process start**. The live S3 settings are written next to the staged database and put back by `ApplyCarriedS3Settings` after migrations, so the archive's S3 settings are never used.

Mutating APIs may return `503` while the maintenance lock is held during export/apply.
