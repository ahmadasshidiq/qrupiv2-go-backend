package export

import (
	"clasenna-go-backend/libs/helpers"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ExportService struct {
	DB *gorm.DB
}

var allowedModels = map[string]bool{
	"users": true, "learning_groups": true, "learning_group_members": true,
	"learning_resources": true, "quizzes": true, "quiz_sessions": true,
	"attendance_logs": true, "attendance_absence_reasons": true,
	"activity_categories": true, "activity_items": true, "activities": true,
	"institutions": true, "roles": true,
}

var institutionScopedModels = map[string]bool{
	"users": true, "learning_groups": true, "learning_group_members": true,
	"learning_resources": true, "quizzes": true, "quiz_sessions": true,
	"attendance_logs": true, "attendance_absence_reasons": true,
	"activity_categories": true, "activity_items": true, "activities": true,
}

func NewExportService(db *gorm.DB) *ExportService {
	return &ExportService{DB: db}
}

func (s *ExportService) buildQuery(dto ExportDTO, filters []FilterDTO) (string, []interface{}) {
	baseTable := dto.Models
	selects := []string{}
	joins := map[string]string{}
	if baseTable == "learning_resources" {
		joins["uploaded_user"] = "join users uploaded_user on uploaded_user.id = learning_resources.uploaded_user_id"
	}
	if baseTable == "attendance_logs" {
		joins["attendance_user"] = "join users attendance_user on attendance_user.id = attendance_logs.user_id"
	}
	if baseTable == "activities" {
		joins["activity_user"] = "join users activity_user on activity_user.id = activities.user_id"
	}
	if baseTable == "quiz_sessions" {
		joins["session_quiz"] = "join quizzes session_quiz on session_quiz.id = quiz_sessions.quiz_id"
	}

	for _, col := range dto.Columns {
		parts := strings.Split(col.Key, ".")

		// CASE 1: Kolom langsung di baseTable
		if len(parts) == 1 && (col.Table == nil || *col.Table == "") {
			selects = append(selects,
				fmt.Sprintf("%s.%s as \"%s\"", baseTable, parts[0], col.Label),
			)
			continue
		}

		// CASE 2: Join tabel lain
		if len(parts) == 1 && col.Table != nil && *col.Table != "" {
			alias := parts[0]
			if col.Alias != nil && *col.Alias != "" {
				alias = *col.Alias
			}

			realTable := *col.Table
			foreignKey := ""
			if col.ForeignKey != nil {
				foreignKey = *col.ForeignKey
			}

			joins[alias] = fmt.Sprintf(
				"left join %s %s on %s.id = %s.%s",
				realTable, alias, alias, baseTable, foreignKey,
			)

			selects = append(selects,
				fmt.Sprintf("%s.name as \"%s\"", alias, col.Label),
			)
			continue
		}

		// CASE 3: Nested join
		previousTable := baseTable
		for i := 0; i < len(parts)-1; i++ {
			currentTable := parts[i]
			joinKey := fmt.Sprintf("%s_%s", previousTable, currentTable)

			foreignKey := fmt.Sprintf("%s_id", helpers.Singular(currentTable))
			if col.ForeignKey != nil {
				foreignKey = *col.ForeignKey
			}

			joins[joinKey] = fmt.Sprintf(
				"left join %s on %s.id = %s.%s",
				currentTable, currentTable, previousTable, foreignKey,
			)

			previousTable = currentTable
		}

		lastTable := parts[len(parts)-2]
		column := parts[len(parts)-1]

		selects = append(selects,
			fmt.Sprintf("%s.%s as \"%s\"", lastTable, column, col.Label),
		)
	}

	query := fmt.Sprintf("select %s from %s", strings.Join(selects, ", "), baseTable)

	for _, join := range joins {
		query += " " + join
	}

	conditions := []string{}
	args := []interface{}{}
	allowedOperators := map[string]bool{"=": true, "!=": true, "like": true, "in": true}
	allowedKeys := map[string]bool{"id": true, "institution_id": true, "user_id": true, "role_id": true, "learning_group_id": true, "activity_item_id": true, "recorded_user_id": true, "quiz_id": true, "name": true, "code": true, "email": true, "type": true, "phone": true, "context_type": true, "context_code": true, "status": true, "barcode": true, "avatar_url": true, "level": true, "major": true, "department": true, "academic_year": true, "title": true, "description": true, "point_value": true, "platform": true, "occurred_at": true, "created_at": true, "updated_at": true, "deleted_at": true}
	for _, filter := range filters {
		if !allowedKeys[filter.Key] {
			continue
		}
		operator := strings.ToLower(filter.Operator)
		if operator == "" {
			operator = "="
		}
		filterTable := baseTable
		if baseTable == "learning_resources" && filter.Key == "institution_id" {
			filterTable = "uploaded_user"
		}
		if baseTable == "quiz_sessions" && filter.Key == "institution_id" {
			filterTable = "session_quiz"
		}
		if baseTable == "attendance_logs" && filter.Key == "institution_id" {
			filterTable = "attendance_user"
		}
		if operator == "is null" {
			conditions = append(conditions, fmt.Sprintf("%s.%s is null", filterTable, filter.Key))
			continue
		}
		if !allowedOperators[operator] {
			continue
		}
		conditions = append(conditions, fmt.Sprintf("%s.%s %s ?", filterTable, filter.Key, operator))
		args = append(args, filter.Value)
	}
	if len(conditions) > 0 {
		query += " where " + strings.Join(conditions, " and ")
	}

	if dto.Limit > 0 && dto.Limit != helpers.UnlimitedLimit {
		query += fmt.Sprintf(" limit %d", dto.Limit)
	}

	return query, args
}

func (s *ExportService) generateExcel(dto ExportDTO, data []map[string]interface{}) (string, error) {
	f := excelize.NewFile()
	sheet := "Sheet1"

	// Title Style
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// Header Style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#010066"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	// Zebra Row Style
	altRowStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#F2F2F2"}, Pattern: 1},
	})

	// Title
	if dto.Title != "" {
		titleRange, _ := excelize.CoordinatesToCellName(1, 1)
		endCell, _ := excelize.CoordinatesToCellName(len(dto.Columns), 1)
		f.MergeCell(sheet, titleRange, endCell)
		f.SetCellValue(sheet, titleRange, dto.Title)
		f.SetCellStyle(sheet, titleRange, endCell, titleStyle)
		f.SetRowHeight(sheet, 1, 28)
	}

	// Header (start from row 2)
	for i, col := range dto.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheet, cell, col.Label)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	// Data (start from row 3)
	for r, row := range data {
		for c, col := range dto.Columns {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+3)
			val := row[col.Label]
			f.SetCellValue(sheet, cell, val)

			// Zebra style
			if r%2 == 1 {
				f.SetCellStyle(sheet, cell, cell, altRowStyle)
			}
		}
	}

	// Auto Width
	for i := range dto.Columns {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, colLetter, colLetter, 20)
	}

	// Freeze header row (row 2)
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       true,
		XSplit:      0,
		YSplit:      2,
		TopLeftCell: "A4",
		ActivePane:  "bottomLeft",
	})

	// Generate filename
	filename := dto.Filename
	if filename == "" {
		filename = fmt.Sprintf("export_%d", time.Now().Unix())
	}
	if filepath.Ext(filename) != ".xlsx" {
		filename += ".xlsx"
	}

	filePath := "/tmp/" + filename
	err := f.SaveAs(filePath)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func (s *ExportService) exportToMap(req ExportDTO, filters []FilterDTO) ([]map[string]interface{}, error) {
	query, args := s.buildQuery(req, filters)

	rows, err := s.DB.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data, err := helpers.ScanRowsToMap(rows)
	println("Data yang di-scan:", data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
