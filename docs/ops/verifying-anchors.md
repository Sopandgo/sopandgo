# Verifying audit anchors

SOPandGO can anchor its audit log with an RFC 3161 Time Stamping Authority (TSA). This page is for the person who checks those anchors: an auditor, a QA lead, or an operator. You need nothing from SOPandGO except the evidence bundle. You can read and run every check yourself.

Operators set anchoring up under **Settings → Audit anchoring**. See [Audit anchoring](deployment.md#audit-anchoring) in the deployment guide.

## What an anchor proves

Every audit row stores a SHA-256 hash of its own content and of the row before it. Changing any row changes every hash after it. Someone with direct database access can still rewrite history and recompute all the hashes, so the chain cannot check itself.

An anchor stops that. At intervals, SOPandGO sends the newest row hash (the *tip*) to the TSA. The TSA signs "this hash existed at time T" and returns a receipt. SOPandGO keeps the receipt and copies it to an S3 bucket. To rewrite history before that row, an attacker would need a receipt from the TSA for the new hash with the old time, and the TSA will not issue one.

An anchor does **not** prove:

- **Anything about rows after the newest receipt.** Rows written since the last anchor can still be deleted without leaving a trace. The interval set under Settings is the size of that window.
- **That history was honest before anchoring was switched on.** The first anchor fixes the history as it stood at that moment. Rows that already failed verification are listed, as accepted by an admin, in the `integrity_baseline` event.
- **Who did something, for rows written before migration v30.** Those rows use hash version 1, which does not cover the actor or entity type (see below).
- **That the TSA is trustworthy.** You trust the TSA's certificate. Get it from the TSA itself.

## Get the evidence bundle

An admin downloads it under **Admin → Integrity → Download evidence bundle**. Each download is recorded in the audit log as `audit_evidence_exported`. The zip contains:

| File | Contents |
| --- | --- |
| `audit_events.jsonl` | Every audit row in chain order, one JSON object per line, with every column the hash covers exactly as stored |
| `anchors/<row>-<hash>.tsr` | One RFC 3161 `TimeStampResp` per anchor (DER) |
| `anchors/<row>-<hash>.json` | Which row and hash that receipt covers (`event_id`, `tip_hash`, `hash_version`, `gen_time`, `tsa_url`, `instance_id`) |
| `tsa-ca.pem` | The TSA CA certificate(s) SOPandGO pinned |
| `README.txt` | A short version of this page |

Copies of the same `.tsr` and `.json` files sit in the operator's S3 bucket under `<prefix>/anchors/<instance_id>/`. If you suspect the server, compare the bundle's receipts with the bucket's. The bucket is the copy an attacker on the server should not be able to change.

## Check it

1. **Check the TSA certificate.** Download your TSA's CA certificate from the TSA's own website or trust list, and compare it with `tsa-ca.pem`. For a qualified EU TSA, the EU Trusted List (published by the European Commission and each member state) names it.

2. **Check each receipt with OpenSSL.** Use the `tip_hash` from the matching `.json`:

   ```sh
   openssl ts -verify -digest <tip_hash> -in anchors/<name>.tsr \
     -CAfile tsa-ca.pem -partial_chain
   ```

   `Verification: OK` means the TSA signed exactly this hash. `-partial_chain` lets the pinned CA be an intermediate rather than a self-signed root.

3. **Recompute the chain and match it to the receipts.** Run [`verify_anchors.py`](verify_anchors.py). It uses only the Python standard library, and it runs step 2 for you when `openssl` is on the `PATH`:

   ```sh
   python3 -I verify_anchors.py sopandgo-audit-evidence-20261118-101500.zip
   ```

   It prints one line per receipt and ends with `RESULT: OK` or `RESULT: PROBLEMS FOUND` (exit status 0 or 1). `python3 -I verify_anchors.py --self-test` checks the script against the reference hashes in the backend tests.

A rewritten row fails in one of two ways. Either the recomputed hash at an anchored row differs from the receipt's `tip_hash`, or someone edited the `.json` to match and OpenSSL reports `message imprint mismatch`.

## The hash algorithm

Each row in `audit_events.jsonl` carries `hash_version`.

**Version 1**, for rows written before migration v30:

```
hash = SHA256( prev_hash || event_type || entity_id || payload || created_at )
```

`||` here is plain string concatenation, with no separators. This version does not cover `entity_type` or `actor_user_id`.

**Version 2**, for every row written since:

```
hash = SHA256( "sopandgo-audit-v2"
               || len(prev_hash)   || prev_hash
               || len(event_type)  || event_type
               || len(entity_type) || entity_type
               || len(entity_id)   || entity_id
               || len(actor)       || actor
               || len(payload)     || payload
               || len(created_at)  || created_at )
```

Each `len(x)` is the length of `x` in UTF-8 bytes, as a 4-byte big-endian unsigned integer. `actor` is `actor_user_id`, or the empty string when it is `null`. Hashes are lowercase hex.

For every row, `prev_hash` must equal the previous row's `hash`. For the first row it must equal 64 `0` characters. `payload` and `created_at` are hashed exactly as they appear in the file. Do not re-encode the JSON payload or reformat the timestamp.

Reference values, also used by the backend tests and `--self-test`, for `prev_hash` = 64 zeros, `event_type` = `user_created`, `entity_type` = `user`, `entity_id` = `1`, `payload` = `{"name":"Alice"}`, `created_at` = `2026-11-18T10:00:00.123456789Z`:

| Version | `actor_user_id` | Hash |
| --- | --- | --- |
| 1 | `user-123` | `5138f3d19128405dad31fba13596f2479534d4cf0b185076fc06442d5ec985f8` |
| 2 | `user-123` | `ec4f0361affac4711a49da38208bf68bdd854b484fd9f36c9ee5e925b075cf2c` |
| 2 | `null` | `bc62abd7a0130c6135a863783c0d29c55a8030a7efd782ce22fa60aa407376dc` |

## Events in the chain that matter here

- **`integrity_baseline`**: written each time anchoring is switched on. Its payload lists the content hash of every SOP version and asset that existed then. It also lists `known_broken`: rows that already failed verification and that the admin accepted. Later SOP versions, assets, and acknowledgments carry their `content_hash` in their own events. So the anchored chain also commits to SOP content.
- **`audit_anchored`**: one per receipt.
- **`audit_anchoring_enabled`**, **`audit_anchoring_disabled`**, **`audit_anchoring_settings_updated`**: switching on, switching off (after a final anchor), and changing the TSA or the interval.
- **`backup_restored`**: a backup restore replaced the chain. Its payload holds the tip that was overwritten (`old_tip_id`, `old_tip_hash`) and the tip of the restored archive (`archive_tip_id`, `archive_tip_hash`). Receipts in S3 for rows after `archive_tip_id` then no longer match. The integrity scan reports them as *replaced by a restore*, for you to review. Ask the operator why the restore happened.

## Limits of the in-app check

**Admin → Integrity** runs the same checks inside SOPandGO. It also compares the bucket with the server: it flags receipts that are in S3 but were removed from the server, and receipts that differ between the two. The in-app check trusts the running server, and the steps above do not. Use the steps above when the server itself is in question.
