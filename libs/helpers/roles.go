package helpers

import "strings"

// RoleName returns the configured database name for a stable application role.
// This keeps authorization logic independent from display names stored in DB.
func RoleName(canonical string) string {
	key := "ROLE_" + strings.ToUpper(strings.ReplaceAll(canonical, "-", "_")) + "_NAME"
	return normalizeRole(ConfigString(key, strings.ReplaceAll(canonical, "_", " ")))
}

func IsRole(actual, canonical string) bool {
	return normalizeRole(actual) == RoleName(canonical)
}

// NormalizeRoleForDashboard exposes the stable role key for response metadata.
func NormalizeRoleForDashboard(value string) string {
	return normalizeRole(value)
}

func normalizeRole(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "-", "_"), " ", "_"))
}
