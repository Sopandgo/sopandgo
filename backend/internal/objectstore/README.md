# Objectstore Package

The port both background jobs use to reach the S3 bucket saved under Settings → Backup: automatic backups stream archives, list them with their timestamps and delete expired ones; audit anchoring copies receipts and reads them back for the integrity scan.

## Pieces

| File | Role |
| --- | --- |
| **`objectstore.go`** | `Store` (put from a reader, bounded get, list with last-modified, delete, Object Lock, root prefix), `Opener`, and `Join` for key prefixes |
| **`s3.go`** | `S3`, the production adapter, built from `S3Config` (bucket, region, endpoint, path style, static credentials or the AWS default chain). Building it does not contact the bucket |
| **`memory.go`** | `Memory`, the in-memory adapter for tests in any package. It can make put, list or delete fail, set the Object Lock flag, and set an object's last-modified time |

## Rules

- `Root` is the key prefix the admin saved. Each job builds its keys below it: backups under `<root>/automated/`, anchoring under `<root>/anchors/<instance>/`.
- A put body that is also an `io.Seeker` (a file, a `bytes.Reader`) is streamed, so a backup archive is never read into memory.
- Object Lock: any answer from the store counts as "not known to be locked". Only network errors fail.
- The opener is `backup.S3SettingsStore.OpenObjectStore`, because the backup package owns those settings. Each job run opens the store once and uses it throughout.
