# Backup Package

Export, validate, and stage-restore of the app data directory, plus optional scheduled S3 uploads.

Operator-facing docs: `docs/ops/backup-and-restore.md`. HTTP surface: `/api/admin/backups/*` and `/api/admin/settings/backup-s3` in `backend/internal/api/api.go` (see `docs/dev/api-reference.md`).

## Pieces

| File | Role |
| --- | --- |
| **`service.go`** | `ExportZip`, `ValidateImportArchive`, `StageImportApply`, maintenance lock, pending restore under `data/_restore_pending/` |
| **`s3_scheduler.go`** | `S3Scheduler`, the automatic S3 backup job: export the same `.zip`, stream it to `<prefix>/automated/`, prune by retention, audit `backup_s3_uploaded`. Timing, retries with backoff, status and failure reporting come from `internal/schedule`; the bucket is reached through `internal/objectstore`. `SaveSettings` validates, saves and audits `backup_s3_settings_updated` in one transaction, then reschedules. `Configure` / `LoadSaved` apply settings without a restart, `RunNow` uploads now, `TestConnection` writes and deletes a probe object |
| **`restore_audit.go`** | `AddStageHook` (other packages carry their own settings across a restore as `carry_*.json`), the chain tip read just before the restore overwrites `app.db`, and `RecordRestoreInAudit`, which writes `backup_restored` after migrations |
| **`s3_settings.go`** | `S3SettingsStore`: the admin-UI settings in `backup_s3_settings` (secret sealed with `SECRET_ENCRYPTION_KEY`), `OpenObjectStore` (the opener both background jobs use, so audit anchoring shares the bucket even when scheduled backups are off), and the carry-over that keeps them across a staged restore |

## Flow (short)

1. **Export** — consistent SQLite snapshot + `sops/` + `manifest.json` into a `.zip`.
2. **Validate** — check archive and schema compatibility without touching live data (`422` on failure via API).
3. **Restore** (`StageImportApply`) — requires confirmation string `RESTORE BACKUP`; stages under `_restore_pending/` and writes a pre-apply snapshot; swap happens on **next process start**. The live S3 settings are written next to the staged database and put back by `ApplyCarriedS3Settings` after migrations, so the archive's S3 settings are never used. Stage hooks (the audit anchoring settings) work the same way. On the first start the live chain tip is read before the swap, and `RecordRestoreInAudit` logs it next to the restored tip, so anchors taken after the archive are explained instead of looking like tampering.

Mutating APIs may return `503` while the maintenance lock is held during export/apply.

## Automatic S3 backups

Each run opens the bucket once and uses it from upload through prune, so one run never spans two buckets. A failed run (upload, prune, or `ErrBusy` from the maintenance lock) is retried after 1, 2, 4… minutes, at most the interval, writes `backup_s3_failed` and sends the `SetOnFailure` notice off the run lock. Retention only ever deletes `sopandgo-automated-*.zip` objects under the automated prefix. Tests use `objectstore.Memory` in place of S3.
