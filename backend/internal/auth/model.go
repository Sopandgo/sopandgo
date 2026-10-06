package auth

import (
	"errors"
	"time"
)

var ErrSelfDisable = errors.New("cannot change own active status")

type User struct {
	ID                 string    `json:"id"`
	DisplayName        string    `json:"display_name"`
	Email              string    `json:"email"`
	Role               string    `json:"role"`
	IsActive           bool      `json:"is_active"`
	MustChangePassword bool      `json:"must_change_password"`
	Locale             string    `json:"locale"`
	CreatedAt          time.Time `json:"created_at"`
}

type Session struct {
	// ID is the refresh-token secret. List responses leave it empty so the
	// admin session API never returns a usable credential.
	ID        string    `json:"id,omitempty"`
	UserID    string    `json:"user_id"`
	UserEmail string    `json:"user_email"`
	UserName  string    `json:"user_name"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type TokenClaims struct {
	UserID   string
	UserRole string
}

// User Roles
const (
	RoleAdmin    = "admin"
	RoleEditor   = "editor"
	RoleApprover = "approver"
	RoleAuditor  = "auditor"
	RoleViewer   = "viewer"
)

// Permission scopes
const (
	ScopeSOPRead         = "sop:read"
	ScopeSOPWrite        = "sop:write"
	ScopeSOPSignReader   = "sop:sign:reader"
	ScopeSOPSignApprover = "sop:sign:approver"
	ScopeSOPSignAuthor   = "sop:sign:author"
	ScopeAdminTools      = "admin:integrity"
	ScopeAuditRead       = "audit:read"
	ScopeTrainingRead    = "training:read"
)
