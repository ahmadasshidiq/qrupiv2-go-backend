package helpers

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type PaginatedMeta struct {
	Filter    map[string]interface{} `json:"filters"`
	Total     int64                  `json:"totalData"`
	Page      int                    `json:"pageSize"`
	Limit     int                    `json:"limit"`
	SortBy    string                 `json:"sortBy,omitempty"`
	SortOrder string                 `json:"sortOrder,omitempty"`
}

type PaginatedResult struct {
	Data []map[string]interface{} `json:"data"`
	Meta PaginatedMeta            `json:"meta"`
}

// UnlimitedLimit is the query limit value used by list endpoints to request
// all matching records.
const UnlimitedLimit = 999

func BuildPaginatedQuery(
	ctx context.Context,
	DB *gorm.DB,
	payload map[string]interface{},
	tableName string,
	baseQuery string,
	queryGroupBy string,
	selectFields string,
	defaultSortBy string,
) (*PaginatedResult, error) {
	// Default values
	if defaultSortBy == "" {
		defaultSortBy = "created_at"
	}

	page := intFromPayload(payload, "page", 1)
	limit := intFromPayload(payload, "limit", 20)
	sortBy := strFromPayload(payload, "sortBy", defaultSortBy)
	sortOrder := strFromPayload(payload, "sortOrder", "desc")

	offset := (page - 1) * limit

	// Extract filters
	filters := make(map[string]interface{})
	for k, v := range payload {
		switch k {
		case "page", "limit", "sortBy", "sortOrder":
			continue
		default:
			filters[k] = v
		}
	}

	res := BuildDynamicWhereClause(filters)

	// Default select fields
	if selectFields == "" {
		selectFields = fmt.Sprintf("%s.*", tableName)
	}

	// Bangun base query
	var stmtQueryBase string
	if baseQuery == "" {
		stmtQueryBase = fmt.Sprintf(`select %s from %s %s`, selectFields, tableName, res.Clause)
	} else {
		stmtQueryBase = fmt.Sprintf(`%s %s %s`, baseQuery, res.Clause, queryGroupBy)
	}

	// base query definition
	pageClause := fmt.Sprintf("limit %d offset %d", limit, offset)
	if limit == UnlimitedLimit {
		pageClause = ""
	}

	stmtQuery := fmt.Sprintf(`
		select * from (
		  %s
		) as row_data
		order by %s %s
		%s
	`, stmtQueryBase, sortBy, sortOrder, pageClause)

	stmtQueryCount := fmt.Sprintf(`
		select count(*) as total_row from (
		  %s
		) as row_data
	`, stmtQueryBase)

	// Query data
	rows, err := DB.Raw(stmtQuery, res.Params...).Rows()
	if err != nil {
		return nil, fmt.Errorf("query data error: %w", err)
	}
	defer rows.Close()

	data, err := ScanRowsToMap(rows)
	if err != nil {
		return nil, fmt.Errorf("scan data error: %w", err)
	}

	// Query total count
	var total int64
	err = DB.Raw(stmtQueryCount, res.Params...).Scan(&total).Error
	if err != nil {
		return nil, fmt.Errorf("count error: %w", err)
	}

	return &PaginatedResult{
		Data: data,
		Meta: PaginatedMeta{
			Filter:    filters,
			Total:     total,
			Page:      page,
			Limit:     limit,
			SortBy:    sortBy,
			SortOrder: sortOrder,
		},
	}, nil
}
