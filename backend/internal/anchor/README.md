# Anchor Package

External anchoring of the audit hash chain. At an interval, a Time Stamping Authority (TSA) timestamps the chain tip (RFC 3161). The receipt is stored in SQLite and copied to the backup S3 bucket. After that, a rewrite of history older than the newest receipt is detectable, even by someone who recomputed every hash.

Operator docs: `docs/ops/deployment.md` (Audit anchoring). Auditor docs and the hash specification: `docs/ops/verifying-anchors.md`. HTTP surface: `/api/admin/audit-anchoring*` in `backend/internal/api/handle_audit_anchoring.go`.

## Pieces

| File | Role |
| --- | --- |
| **`service.go`** | `Service`: one run (`anchorOnce`) as a job of `internal/schedule` (`Start`, `AnchorNow`, `TriggerSoon` → `RunSoon`), and the settings flows `Probe` and `Save` (switch on, change, switch off) |
| **`tsa.go`** | RFC 3161 client (`TSA.Stamp`, via `digitorus/timestamp`), `VerifyReceipt` (digest, CMS signature, signer chains to the pinned CA, RFC 3161 key usage), `PinFromToken` |
| **`verify.go`** | `Status`, and `Check`: the anchoring part of the admin integrity scan |
| **`baseline.go`** | The `integrity_baseline` payload (all SOP version and asset hashes, plus rows accepted as already broken) |
| **`bundle.go`** | `WriteBundle`: the evidence zip for auditors |
| **`store.go`** | `audit_anchoring_settings` (one row, TSA secret sealed with `SECRET_ENCRYPTION_KEY`), `audit_anchors`, and the restore carry-over |

## A run

1. Copy any receipts not yet in S3.
2. `audit.VerifySince(last anchor)`: the anchored row must still have its hash, and every later row must link to the one before and recompute. Rows an admin accepted in a baseline are ignored. Any other failure stops the run: a chain that changed is never anchored.
3. Skip if nothing but `audit_anchored` events was logged since the last anchor.
4. Ask the TSA to timestamp the raw tip hash (SHA-256, with a nonce and `certReq`). Check the response with `VerifyReceipt` against the pinned CA.
5. Insert into `audit_anchors`, which also records the CA used. Put `<prefix>/anchors/<instance>/<row>-<hash>.tsr` and `.json` in S3. Log `audit_anchored`.

Receipts reach S3 through the `internal/objectstore` port, opened once per run with `backup.S3SettingsStore.OpenObjectStore`; tests use `objectstore.Memory`. The schedule, backoff and failure reporting belong to `internal/schedule`: failures are audited (`audit_anchor_failed`), sent to `SetOnFailure` after the run lock is released, and retried after 1, 2, 4… minutes, at most the interval. A run that saved a receipt but could not copy it still counts as failed. Audit writes are never blocked.

## Confirmation protocol

`Save` with a new TSA, a re-pin, or a switch-on returns `*NeedsConfirmation` with a `ProbeResult`: the CA to pin, its fingerprint, Object Lock state, and rows that already fail verification. The caller repeats the request with `ConfirmFingerprint` and, when switching on, `AcceptBroken` equal to those rows. The server probes again, so a changed TSA or a new broken row asks again.

## Invariants

- Receipts are only accepted when they chain to a pinned CA and carry the critical, time-stamping-only extended key usage that RFC 3161 requires (OpenSSL enforces the same rule).
- Every settings change writes its audit event in the same transaction as the settings: switch-on (`integrity_baseline` and `audit_anchoring_enabled`), change (`audit_anchoring_updated`) and switch-off (`audit_anchoring_disabled`). Switching off takes a final anchor first. All three notify through `SetOnChange` after the commit.
- A restore keeps the running instance's settings (`WriteCarryOver` as a backup stage hook, then `ApplyCarriedSettings`). The receipts table is not carried; `Check` explains receipts in S3 with the `backup_restored` event.
