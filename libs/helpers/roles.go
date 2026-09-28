package helpers

import (
	"os"
	"strings"
)

// RoleName returns the configured database name for a stable application role.
// This keeps authorization logic independent from display names stored in DB.
func RoleName(canonical string) string {
	key := "ROLE_" + strings.ToUpper(strings.ReplaceAll(canonical, "-", "_")) + "_NAME"
	return normalizeRole(ConfigString(key, strings.ReplaceAll(canonical, "_", " ")))
}

func IsRole(actual, canonical string) bool {
	// Accept both the stable application key (institution_admin) and the
	// configured/display name (Admin Institusi). JWTs and older services do
	// not always carry the same representation.
	actualRole := normalizeRole(actual)
	canonicalRole := normalizeRole(canonical)
	if actualRole == canonicalRole || actualRole == RoleName(canonical) {
		return true
	}
	// Role names in existing databases are display labels. Keep these aliases
	// stable even when a service is started without loading the root .env.
	aliases := map[string][]string{
		"super_admin":       {"super_admin", "superadmin", "super_admin", "super_admin"},
		"institution_admin": {"institution_admin", "admin_institusi", "admin_sekolah", "school_admin"},
		"instructor":        {"instructor", "instruktur", "guru", "teacher"},
		"student":           {"student", "pelajar", "siswa"},
		"dinas_pendidikan":  {"dinas_pendidikan", "dinas_pendidikan"},
	}
	for _, alias := range aliases[canonicalRole] {
		if actualRole == alias {
			return true
		}
	}
	return false
}

// IsRoleID compares the immutable role UUID from JWT claims. The UUID is the
// preferred authorization identity; name matching remains only as a legacy
// fallback until all ROLE_*_ID values are configured.
func IsRoleID(actualID, canonical string) bool {
	expectedID := os.Getenv("ROLE_" + strings.ToUpper(strings.ReplaceAll(canonical, "-", "_")) + "_ID")
	return expectedID != "" && strings.EqualFold(strings.TrimSpace(actualID), strings.TrimSpace(expectedID))
}

func RoleKeyFromID(actualID string) string {
	for _, role := range []string{"super_admin", "institution_admin", "instructor", "student", "dinas_pendidikan"} {
		if IsRoleID(actualID, role) {
			return role
		}
	}
	return ""
}

func normalizeRole(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), "-", "_"), " ", "_"))
}
