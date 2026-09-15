package export

import "github.com/google/uuid"

type ColumnDTO struct {
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	Table      *string `json:"table,omitempty"`
	ForeignKey *string `json:"foreignKey,omitempty"`
	Alias      *string `json:"alias,omitempty"`
}

type FilterDTO struct {
	Key      string `json:"key"`
	Operator string `json:"operator,omitempty"`
	Value    any    `json:"value"`
}

type ExportDTO struct {
	Title    string      `json:"title"`
	Filename string      `json:"filename"`
	Models   string      `json:"models"`
	PageSize int         `json:"pageSize"`
	Limit    int         `json:"limit"`
	Columns  []ColumnDTO `json:"column"`
	Filters  []FilterDTO `json:"filters,omitempty"`
}

type ExportQCSectionDTO struct {
	Title        string    `json:"title"`
	Filename     string    `json:"filename"`
	ChangeOverID uuid.UUID `json:"change_over_id"`
}
