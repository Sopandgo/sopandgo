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
