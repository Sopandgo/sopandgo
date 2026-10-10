# Auth Package Architecture

This package manages users, sessions (refresh tokens), password resets, and RBAC scopes.
It follows a **struct-based service pattern** so all business logic goes through `auth.Service`.

## Core Principles

1. **The Service Struct (`auth.Service`):**
   * **Single entry point** for business logic; holds `*sql.DB`, `*audit.Logger`, and `dataDir` for profile pictures.
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
| **`service.go`** | Constructor | `NewService(db, logger, dataDir)` |
| **`user_service.go`** | Users and password reset | `RegisterUser`, `UpdateUserRole`, `SetUserActiveStatus`, `UpdateUserPassword`, `ChangeOwnPassword`, `GeneratePasswordResetToken`, `ResetPassword` |
| **`avatar_service.go`** | Profile pictures on disk | `SetAvatar`, `RemoveAvatar`, `AvatarAbsolutePath` |
| **`avatar_image.go`** | Decode / crop / resize / JPEG | `processAvatarSizes` (96/256/512/1024) |
| **`session_service.go`** | Refresh sessions | `CreateSession`, `ValidateSession`, `RevokeSession`, `RevokeOtherUserSessions`, `RevokeAllUserSessions`, `RevokeAllSessions`, `StartSessionsCleanupTask` |
| **`store.go`** | Unexported SQL | `createUserRecord`, `getSessionRecord`, … |
| **`password.go`** | Hashing / rules | `HashPassword`, `ValidatePassword` |
| **`token.go`** | PASETO access tokens | `GenerateAccessToken` |
| **`rbac.go`** | Role → scopes | `HasScope` |
| **`bootstrap.go`** | Default admin | `EnsureAdminUser` |
| **`model.go`** | Types and constants | `User` (includes `has_avatar`, `avatar_content_hash`), `Session`, `ScopeAdminTools` (`admin:integrity`) |

## Usage Examples

### Initialization

```go
authService := auth.NewService(db, auditLogger, dataDir)
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

### Signed-in password change

`ChangeOwnPassword` verifies the current password before calling `UpdateUserPassword`, and returns `ErrWrongCurrentPassword` otherwise. Reset and bootstrap call `UpdateUserPassword` directly.

```go
err := authService.ChangeOwnPassword(userID, "CurrentPassword123!", "NewStrongPassword123!")
```

### Sessions

```go
refreshToken, err := authService.CreateSession(userID)
userID, role, err := authService.ValidateSession(refreshToken)
revoked, err := authService.RevokeOtherUserSessions(userID, refreshToken) // keeps refreshToken
authService.StartSessionsCleanupTask(ctx, 24*time.Hour)
```
