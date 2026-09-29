# Auth Package Architecture

This package manages users, sessions (refresh tokens), password resets, and RBAC scopes.
It follows a **struct-based service pattern** so all business logic goes through `auth.Service`.

## Core Principles

1. **The Service Struct (`auth.Service`):**
   * **Single entry point** for business logic; holds `*sql.DB` and `*audit.Logger`.
   * **Concurrency:** Internal `sync.RWMutex` for critical writes.
   * **Transactions:** Owns `Begin` / `Rollback` / `Commit`.
   * **Audit:** Writes compliance logs in the same transaction as the data change.

2. **Store Layer (`store.go`):**
   * **Unexported** helpers (e.g. `createUserRecord`). Only the service can touch SQL.
   * Raw queries only; no business logic or mutexes.
   * Accepts `audit.DBTX` (`*sql.DB` or `*sql.Tx`).

3. **Model (`model.go`):**
   * Structs, roles (`admin`, `approver`, `editor`, `viewer`, `auditor`), and scopes (e.g. `sop:write`, `admin:integrity`). The `auditor` role includes `audit:read`, but audit-log HTTP APIs currently require `admin:integrity` — see `docs/dev/auth-architecture.md`.

Login (password verify + session create) is orchestrated in the **API** layer (`handle_auth.go`) using `GetUserByEmail`, bcrypt compare, and `CreateSession`. There is no `Login` method on `Service`.

## File Structure

| File | Purpose | Key methods |
| --- | --- | --- |
| **`service.go`** | Constructor | `NewService` |
| **`user_service.go`** | Users and password reset | `RegisterUser`, `UpdateUserRole`, `SetUserActiveStatus`, `UpdateUserPassword`, `GeneratePasswordResetToken`, `ResetPassword` |
| **`session_service.go`** | Refresh sessions | `CreateSession`, `ValidateSession`, `RevokeSession`, `RevokeAllUserSessions`, `RevokeAllSessions`, `StartSessionsCleanupTask` |
| **`store.go`** | Unexported SQL | `createUserRecord`, `getSessionRecord`, … |
| **`password.go`** | Hashing / rules | `HashPassword`, `ValidatePassword` |
| **`token.go`** | PASETO access tokens | `GenerateAccessToken` |
| **`rbac.go`** | Role → scopes | `HasScope` |
| **`bootstrap.go`** | Default admin | `EnsureAdminUser` |
| **`model.go`** | Types and constants | `User`, `Session`, `ScopeAdminTools` (`admin:integrity`) |

## Usage Examples

### Initialization

```go
authService := auth.NewService(db, auditLogger)
```

### Registering a user (invite-only)

Admins never set a password. `RegisterUser` stores an unknowable random hash; the user sets a password via invite/reset token.

```go
id, err := authService.RegisterUser(
    "John Doe",
    "john@example.com",
    auth.RoleViewer,
    &adminID,
)
```

### Password reset / invite

```go
token, user, err := authService.GeneratePasswordResetToken(userID)
err = authService.ResetPassword(token, "NewStrongPassword123!")
```

### Sessions

```go
refreshToken, err := authService.CreateSession(userID)
userID, role, err := authService.ValidateSession(refreshToken)
authService.StartSessionsCleanupTask(ctx, 24*time.Hour)
```
