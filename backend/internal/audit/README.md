# Audit Package Architecture

This package records and retrieves system audit logs with a **tamper-evident hash chain**.

It is designed to participate in transactions owned by other packages (`auth`, `sop`, `mail`, …).

## Core Principles

1. **Service Layer (`audit.go`):**
   * **`Log`:** Accepts an `executor` (`DBTX`) so it can join an existing transaction. Marshals any payload to JSON. Normalizes `event_type` / `entity_type` to lowercase. Optional strict mode via `AUDIT_STRICT_TYPES=true`.
   * **`List`:** Returns events with per-row `HashValid`, plus a total count. Filters via `ListFilters`.
   * **`ListFilterOptions`:** Distinct event/entity types for admin UIs.

2. **Store Layer (`store.go`):**
   * Raw INSERT/SELECT; unexported `*Record` helpers.
   * On insert, reads the previous chain tip and computes the new hash.
   * Reading the tip and inserting must happen under one write lock, or two writers can link to the same tip and fork the chain.
   * **Rule:** every transaction holds the write lock from begin. `storage.Open` opens every connection with immediate transactions, so a caller's transaction already holds the lock when it calls `Log`, whatever it did before. `Log(nil, …)` and `Log(db, …)` open an ordinary transaction on the shared handle, insert, and commit.
   * A writer waits up to `DB_BUSY_TIMEOUT` (default `5s`, set by the storage package) for another writer, process, or transaction. Timeout errors are returned to the caller and must be surfaced.
   * The logger relies on a handle opened by `storage.Open` or `storage.OpenInMemory`. A handle opened another way may lack these settings.

3. **Model (`model.go`):**
   * `AuditEvent`, `ListFilters`, `AuditEventWithVerification`.

## File Structure

| File | Purpose | Key functions |
| --- | --- | --- |
| **`audit.go`** | Logger API | `New`, `Log`, `List`, `ListFilterOptions` |
| **`store.go`** | SQL + hashing | `insertEventRecord`, `listEventsRecord` |
| **`model.go`** | Types | `AuditEvent`, `ListFilters` |

## Usage Examples

### Writing (transactional)

```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback()

err = auditLogger.Log(
    tx,
    "user_registered",
    "user",
    userID,
    &actorID,
    map[string]string{
        "email": email,
        "role":  "editor",
    },
)
if err != nil {
    return err
}
return tx.Commit()
```

### Reading

```go
events, total, err := auditLogger.List(10, 0, audit.ListFilters{
    EventType: "user_registered",
})
for _, e := range events {
    if !e.HashValid {
        log.Printf("ALERT: Tampering detected in record %d", e.ID)
    }
}
```

The HTTP API only **lists** logs (`GET /api/admin/audit-logs`). There is no public insert endpoint.

## Integrity Algorithm

At insert time:

`CurrentHash = SHA256( PrevHash + EventType + EntityID + PayloadJSON + CreatedAt )`

The record **ID is not** part of the hash input. The first record uses a fixed genesis `PrevHash`.
