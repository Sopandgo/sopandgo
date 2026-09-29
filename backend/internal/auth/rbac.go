package auth

import "slices"

// Roles to permission scopes map
var roleScopes = map[string][]string{
	RoleAdmin:    {ScopeSOPRead, ScopeSOPWrite, ScopeAdminTools, ScopeAuditRead, ScopeTrainingRead, ScopeSOPSignAuthor, ScopeSOPSignApprover, ScopeSOPSignReader},
	RoleApprover: {ScopeSOPRead, ScopeSOPWrite, ScopeTrainingRead, ScopeSOPSignAuthor, ScopeSOPSignApprover, ScopeSOPSignReader},
	RoleEditor:   {ScopeSOPRead, ScopeSOPWrite, ScopeSOPSignAuthor, ScopeSOPSignReader},
	RoleAuditor:  {ScopeSOPRead, ScopeAuditRead},
	RoleViewer:   {ScopeSOPRead, ScopeSOPSignReader},
}

func HasPermission(userRole string, requiredScope string) bool {
	// Admin always has all permissions
	if userRole == RoleAdmin {
		return true
	}

	// Retrieve scopes for the given role
	scopes, exists := roleScopes[userRole]
	if !exists {
		return false
	}

	// Check if the required scope exists in this role's list
	return slices.Contains(scopes, requiredScope)
}
