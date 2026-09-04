package export

import "github.com/google/uuid"

type ColumnDTO struct {
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	Table      *string `json:"table,omitempty"`
	ForeignKey *string `json:"foreignKey,omitempty"`
	Alias      *string `json:"alias,omitempty"`
}

type ExportDTO struct {
	Title    string      `json:"title"`
	Filename string      `json:"filename"`
	Models   string      `json:"models"`
	PageSize int         `json:"pageSize"`
	Limit    int         `json:"limit"`
	Columns  []ColumnDTO `json:"column"`
}

type ExportQCSectionDTO struct {
	Title        string    `json:"title"`
	Filename     string    `json:"filename"`
	ChangeOverID uuid.UUID `json:"change_over_id"`
}
