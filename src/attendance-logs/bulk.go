package attendance_logs

import (
	"fmt"

	"clasenna-go-backend/libs/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BulkValidationError struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

func (e *BulkValidationError) Error() string {
	return fmt.Sprintf("attendance_logs[%d]: %s", e.Index, e.Message)
}

func (s *AttendanceLogService) createBulk(ctx *gin.Context, dto BulkCreateDTO) (*BulkCreateResult, error) {
	if len(dto.AttendanceLogs) == 0 || len(dto.AttendanceLogs) > MaxBulkAttendanceLogs {
		return nil, &BulkValidationError{
			Index:   -1,
			Message: fmt.Sprintf("attendance_logs must contain between 1 and %d records", MaxBulkAttendanceLogs),
		}
	}

	created := make([]models.AttendanceLog, 0, len(dto.AttendanceLogs))
	err := s.DB.WithContext(ctx.Request.Context()).Transaction(func(tx *gorm.DB) error {
		txService := &AttendanceLogService{DB: tx}
		for index, entry := range dto.AttendanceLogs {
			attendance, err := txService.create(ctx, entry)
			if err != nil {
				return &BulkValidationError{Index: index, Message: err.Error()}
			}
			created = append(created, *attendance)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(created))
	for index := range created {
		ids[index] = created[index].ID.String()
	}
	return &BulkCreateResult{Count: len(created), IDs: ids}, nil
}

var _ error = (*BulkValidationError)(nil)
