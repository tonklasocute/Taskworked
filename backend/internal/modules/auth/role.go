package auth

// IsOrgAdmin reports whether a role has org-wide administrative
// privileges (bypasses per-resource ownership checks in other modules).
func IsOrgAdmin(r Role) bool {
	return r == RoleAdmin || r == RoleSuperAdmin
}
