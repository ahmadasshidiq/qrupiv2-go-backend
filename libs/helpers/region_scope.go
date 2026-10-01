package helpers

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// ApplyRegionScope adds an authenticated user's region restriction to a
// paginated query. The region is taken from JWT middleware, never from the
// request query string.
func ApplyRegionScope(ctx *gin.Context, params map[string]interface{}, tableAlias string) error {
	level, code := ctx.GetString("region_level"), ctx.GetString("region_code")
	if level == "" || code == "" || IsRoleID(ctx.GetString("role_id"), "super_admin") {
		return nil
	}
	column := map[string]string{
		"province": "province_code",
		"regency":  "regency_code",
		"district": "district_code",
		"village":  "village_code",
	}[level]
	if column == "" {
		return fmt.Errorf("invalid region scope")
	}
	params[tableAlias+"."+column] = code
	return nil
}
