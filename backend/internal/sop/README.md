# SOP Package Architecture

This package manages Standard Operating Procedures (SOPs), version history and lifecycle, tags, favorites, assets, acknowledgments, PDF artifacts, diffs, and training coverage.

It follows a **Service Layer Pattern** (`sop.Service`) for data consistency, atomic audit logging, and secure file handling.

## Core Principles

1. **Service Layer (`*_service.go` and related):**
   * **Owns the Transaction:** Calls `db.Begin()`, `tx.Rollback()`, and `tx.Commit()`.
   * **Owns the Filesystem:** Handles file I/O using secure paths. Cleans up files if the DB commit fails.
   * **Owns the Audit Log:** Calls `audit.Log` inside the transaction.
   * **Orchestrates:** Calls private store functions to read/write DB records.

2. **Store Layer (`store.go`, `tag_store.go`):**
   * **Simple SQL:** Executes queries and maps rows to structs.
   * **Flexible:** Accepts `audit.DBTX` (works with `*sql.DB` and `*sql.Tx`).
   * Store helpers are package-private; callers use `*Service` methods.

3. **Hybrid Storage Strategy:**
   * **Database:** Relative paths (e.g. `sops/123/v1.md`) for portability, plus metadata, tags, favorites, lifecycle states, and PDF artifact rows.
   * **Disk:** Absolute paths under `DATA_DIR`, resolved via `SafeJoin`.
   * **Network:** Streams files with `http.ServeFile` from the API layer.

4. **Version lifecycle:** Content is append-only. Status moves `draft` → `rc` → `published` (or `rejected`); prior published versions become `superseded`. See `docs/concepts/versioning-and-signatures.md`.

## File Structure

| File | Purpose | Key methods |
| --- | --- | --- |
| **`service.go`** | `Service` constructor and PDF config | `NewService`, `IsPDFExportEnabled` |
| **`sop_service.go`** | SOP containers; list rows carry tags, favorite state, the latest version (number, state, date) and the published version number | `RegisterSOP`, `ListSOPs`, `GetSOPByIDWithTags` |
| **`version_service.go`** | Drafts, content I/O, lifecycle | `RegisterSOPVersion`, `TransitionVersionState`, `ApproveSOPVersion`, summaries, integrity |
| **`acknowledgment_service.go`** | Reader sign-off on published versions, and signature status | `AddAcknowledgment`, `GetSignatureStatusByUser` |
| **`asset_service.go`** | Binary uploads | `AddAsset`, `GetAssetPath`, `VerifyAssetIntegrity` |
| **`favorite_service.go`** | Per-user bookmarks | `FavoriteSOP`, `UnfavoriteSOP` |
| **`tag_service.go`** | Global tags + attach/detach | `CreateTag`, `AttachTagToSOP`, `SetTagStatus` |
| **`pdf_artifact_service.go`** | Gotenberg PDF artifacts | `GetVersionPDFArtifactPath`, backfill helpers |
| **`diff.go` / `publish_activity.go`** | Line diffs and recent publishes | `DiffSOPVersion`, `ListRecentPublishes` |
| **`training.go`** | Reader-signature coverage | `TrainingCoverage` |
| **`integrity_service.go`** | System-wide scan | `RunSystemIntegrityCheck` |
| **`validate.go`** | Markdown asset references | `ValidateSOPContentAssets` |
| **`store.go` / `tag_store.go`** | Raw SQL | private `*Record` helpers |
| **`model.go`** | Types and state constants | `SOP`, `SOPVersion`, `StateDraft`, … |

## Usage Examples

### 1. Creating a draft version

```go
id, ver, err := sopService.RegisterSOPVersion(sopID, content, changeSummary, &actorID)
```

### 2. Promote / approve

```go
err := sopService.TransitionVersionState(versionID, sop.StateRC, actorID)
ackID, err := sopService.ApproveSOPVersion(versionID, approverID)
```

### 3. Integrity

```go
valid, err := sopService.VerifyVersionIntegrity(versionID)
```
