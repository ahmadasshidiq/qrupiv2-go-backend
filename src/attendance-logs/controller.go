package attendance_logs

import (
	"clasenna-go-backend/libs/helpers"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AttendanceLogController struct{ Service *AttendanceLogService }

func NewController(service *AttendanceLogService) *AttendanceLogController {
	return &AttendanceLogController{Service: service}
}

// @Summary Get all Attendance Logs
// @Tags Attendance Log API
// @Param sortBy query string false "Sort by field"
// @Param sortOrder query string false "Sort order" Enums(asc, desc)
// @Param limit query int false "Number of results per page"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /attendance-logs [get]
func (c *AttendanceLogController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-logs", http.StatusOK, data)
}

// @Summary Get Attendance Log by ID
// @Tags Attendance Log API
// @Param id path string true "Attendance Log ID"
// @Security BearerAuth
// @Router /attendance-logs/{id} [get]
func (c *AttendanceLogController) GetByID(ctx *gin.Context) {
	data, err := c.Service.getByID(ctx, ctx.Param("id"))
	if err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "attendance-logs", http.StatusNotFound, "Attendance Log not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-logs", http.StatusOK, data)
}

// @Summary Create Attendance Log
// @Tags Attendance Log API
// @Param dto body CreateDTO true "Attendance Log Data"
// @Security BearerAuth
// @Router /attendance-logs [post]
func (c *AttendanceLogController) Create(ctx *gin.Context) {
	var dto CreateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.create(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-logs", http.StatusCreated, data.ID)
}

// @Summary Create Attendance Logs in bulk
// @Description Creates up to 200 attendance logs atomically; all records are rolled back when one record is invalid
// @Tags Attendance Log API
// @Param dto body BulkCreateDTO true "Bulk Attendance Log Data"
// @Security BearerAuth
// @Router /attendance-logs/bulk [post]
func (c *AttendanceLogController) CreateBulk(ctx *gin.Context) {
	var dto BulkCreateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusBadRequest, err)
		return
	}
	result, err := c.Service.createBulk(ctx, dto)
	if err != nil {
		var validationError *BulkValidationError
		if errors.As(err, &validationError) {
			helpers.RespondErrorData(ctx, "attendance-logs", http.StatusBadRequest, validationError, err)
			return
		}
		helpers.RespondError(ctx, "attendance-logs", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-logs", http.StatusCreated, result)
}

// @Summary Check in Attendance
// @Tags Attendance Log API
// @Param dto body CheckInDTO true "Attendance Check-in Data"
// @Security BearerAuth
// @Router /attendance-logs/check-in [post]
func (c *AttendanceLogController) CheckIn(ctx *gin.Context) {
	var dto CheckInDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.checkIn(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusBadRequest, err)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-logs", http.StatusCreated, data.ID)
}

// @Summary Check out Attendance
// @Tags Attendance Log API
// @Param id path string true "Attendance Log ID"
// @Param dto body CheckOutDTO true "Attendance Check-out Data"
// @Security BearerAuth
// @Router /attendance-logs/{id}/check-out [put]
func (c *AttendanceLogController) CheckOut(ctx *gin.Context) {
	var dto CheckOutDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.checkOut(ctx, ctx.Param("id"), dto)
	if err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusBadRequest, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "attendance-logs", http.StatusNotFound, "Attendance Log not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-logs", http.StatusOK, data.ID)
}

// @Summary Update Attendance Log
// @Tags Attendance Log API
// @Param id path string true "Attendance Log ID"
// @Param dto body UpdateDTO true "Attendance Log Data"
// @Security BearerAuth
// @Router /attendance-logs/{id} [put]
func (c *AttendanceLogController) Update(ctx *gin.Context) {
	var dto UpdateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.update(ctx, ctx.Param("id"), dto)
	if err != nil {
		helpers.RespondError(ctx, "attendance-logs", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "attendance-logs", http.StatusNotFound, "Attendance Log not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-logs", http.StatusOK, data.ID)
}

// @Summary Archive Attendance Log
// @Tags Attendance Log API
// @Param id path string true "Attendance Log ID"
// @Security BearerAuth
// @Router /attendance-logs/{id}/archived [delete]
func (c *AttendanceLogController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "attendance-logs", func(id string) (bool, error) { return c.Service.archive(ctx, id) })
}

// @Summary Delete Attendance Log permanently
// @Tags Attendance Log API
// @Param id path string true "Attendance Log ID"
// @Security BearerAuth
// @Router /attendance-logs/{id} [delete]
func (c *AttendanceLogController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "attendance-logs", func(id string) (bool, error) { return c.Service.delete(ctx, id) })
}
