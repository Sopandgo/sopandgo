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

4. **Version lifecycle:** Content is append-only. Status moves `draft` → `rc` → `published`, or `rc` → `rejected`; prior published versions become `superseded`. The rules live in one table in `lifecycle.go`. A wrong state is an `errkind.Conflict` error and an unknown version an `errkind.NotFound` error (see `backend/internal/errkind`). A whole SOP can be archived, which freezes all of this until it is unarchived. See `docs/concepts/versioning-and-signatures.md`.

## File Structure

| File | Purpose | Key methods |
| --- | --- | --- |
| **`service.go`** | `Service` constructor and PDF config | `NewService`, `IsPDFExportEnabled` |
| **`archive_service.go`** | Archive and unarchive an SOP (reason required, blocked while a release candidate is pending). Archived SOPs leave the default list, signature status and team coverage; every write to their content, lifecycle, signatures and assets returns `ErrSOPArchived` via `requireSOPActiveRecord` inside the write's transaction | `ArchiveSOP`, `UnarchiveSOP` |
| **`sop_service.go`** | SOP containers; list rows carry tags, favorite state, the latest version (id, number, state, date) and the published version number; the detail adds the published version's id. Both are derived from the version states on read, not stored | `RegisterSOP`, `ListSOPs`, `GetSOPByIDWithTags` |
| **`lifecycle.go`** | The lifecycle module, private to the package. One table lists every allowed transition (new version → draft, draft → rc, rc → rejected, rc → published, reader acknowledgment on published) with its side effects. `runTransition` is the only code that opens a transaction for a lifecycle change: it loads the version, refuses an archived SOP, checks the table, writes the state row, acknowledgment and audit events, commits, then generates the PDF. The actor is required | `runTransition` (private) |
| **`version_service.go`** | Drafts and content I/O, and the version operations, each a request for a transition; a rejection stores its reason on the state row and in the audit event, and the summary returns it | `RegisterSOPVersion`, `PromoteSOPVersion`, `ApproveSOPVersion`, `RejectSOPVersion`, summaries, integrity |
| **`acknowledgment_service.go`** | Reader sign-off on published versions (a transition request), and signature status | `AddAcknowledgment`, `GetSignatureStatusByUser` |
| **`asset_service.go`** | Binary uploads | `AddAsset`, `GetAssetPath`, `VerifyAssetIntegrity` |
| **`favorite_service.go`** | Per-user bookmarks | `FavoriteSOP`, `UnfavoriteSOP` |
| **`tag_service.go`** | Global tags + attach/detach | `CreateTag`, `AttachTagToSOP`, `SetTagStatus` |
| **`pdf_artifact_service.go`** | Gotenberg PDF artifacts, generated after the lifecycle commit; failures are logged and left to the backfill, never returned to the caller | `GetVersionPDFArtifactPath`, `BackfillPDFArtifactsForGenerator` |
| **`diff.go` / `publish_activity.go`** | Line diffs and recent publishes | `DiffSOPVersion`, `ListRecentPublishes` |
| **`training.go`** | Reader-signature coverage | `TrainingCoverage` |
| **`integrity_service.go`** | System-wide scan | `RunSystemIntegrityCheck` |
| **`validate.go`** | Markdown asset references | `ValidateSOPContentAssets` |
| **`store.go` / `tag_store.go`** | Raw SQL | private `*Record` helpers |
| **`model.go`** | Types and state constants | `SOP`, `SOPVersion`, `StateDraft`, … |

## Usage Examples

### 1. Creating a draft version

```go
id, ver, err := sopService.RegisterSOPVersion(sopID, content, changeSummary, actorID) // actor required
```

### 2. Promote / approve

```go
err := sopService.PromoteSOPVersion(versionID, actorID)
ackID, err := sopService.ApproveSOPVersion(versionID, approverID)
err = sopService.RejectSOPVersion(versionID, approverID, reason) // reason required, stored
```

### 3. Archive

```go
err := sopService.ArchiveSOP(sopID, approverID, reason) // reason required, stored
err = sopService.UnarchiveSOP(sopID, approverID)
list, total, err := sopService.ListSOPs(userID, 50, 0, "", "", false, false, sop.ArchiveFilterOnly)
```

### 4. Integrity

```go
valid, err := sopService.VerifyVersionIntegrity(versionID)
```
