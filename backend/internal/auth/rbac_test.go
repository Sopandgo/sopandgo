package auth_test

import (
	"testing"

	"github.com/sopandgo/sopandgo/backend/internal/auth"
)

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name          string
		role          string
		requiredScope string
		expected      bool
	}{
		// 1. Admin Override Tests
		// Admins should bypass the map entirely and be granted everything
		{"Admin has explicit scope", auth.RoleAdmin, auth.ScopeAdminTools, true},
		{"Admin has fake/unknown scope", auth.RoleAdmin, "scope:launch_nukes", true},

		// 2. Valid Role, Valid Scope (Happy Paths)
		{"Approver can approve", auth.RoleApprover, auth.ScopeSOPSignApprover, true},
		{"Editor can author", auth.RoleEditor, auth.ScopeSOPSignAuthor, true},
		{"Viewer can read", auth.RoleViewer, auth.ScopeSOPRead, true},
		{"Auditor can read audits", auth.RoleAuditor, auth.ScopeAuditRead, true},

		// 3. Valid Role, Invalid Scope (Security Denials)
		{"Viewer cannot write SOPs", auth.RoleViewer, auth.ScopeSOPWrite, false},
		{"Editor cannot approve SOPs", auth.RoleEditor, auth.ScopeSOPSignApprover, false},
		{"Auditor cannot write SOPs", auth.RoleAuditor, auth.ScopeSOPWrite, false},
		{"Approver cannot access admin tools", auth.RoleApprover, auth.ScopeAdminTools, false},
		{"Approver can read training coverage", auth.RoleApprover, auth.ScopeTrainingRead, true},
		{"Editor cannot read training coverage", auth.RoleEditor, auth.ScopeTrainingRead, false},

		// 4. Unknown or Blank Roles (Fallback Security)
		{"Unknown role is denied", "super-hacker", auth.ScopeSOPRead, false},
		{"Empty role string is denied", "", auth.ScopeSOPRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.HasPermission(tt.role, tt.requiredScope)
			if result != tt.expected {
				t.Errorf("HasPermission(%q, %q) = %v; expected %v", tt.role, tt.requiredScope, result, tt.expected)
			}
		})
	}
}
