package attendance_absence_reasons

import (
	"net/http"

	"clasenna-go-backend/libs/helpers"
	"github.com/gin-gonic/gin"
)

type AttendanceAbsenceReasonController struct {
	Service *AttendanceAbsenceReasonService
}

func NewController(service *AttendanceAbsenceReasonService) *AttendanceAbsenceReasonController {
	return &AttendanceAbsenceReasonController{Service: service}
}

// @Summary Get all Attendance Absence Reasons
// @Tags Attendance Absence Reason API
// @Param sortBy query string false "Sort by field"
// @Param sortOrder query string false "Sort order" Enums(asc, desc)
// @Param limit query int false "Number of results per page"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /attendance-absence-reasons [get]
func (c *AttendanceAbsenceReasonController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "attendance-absence-reasons", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "attendance-absence-reasons", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-absence-reasons", http.StatusOK, data)
}

// @Summary Get Attendance Absence Reason by ID
// @Tags Attendance Absence Reason API
// @Param id path string true "Attendance Absence Reason ID"
// @Security BearerAuth
// @Router /attendance-absence-reasons/{id} [get]
func (c *AttendanceAbsenceReasonController) GetByID(ctx *gin.Context) {
	data, err := c.Service.getByID(ctx, ctx.Param("id"))
	if err != nil {
		helpers.RespondError(ctx, "attendance-absence-reasons", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondError(ctx, "attendance-absence-reasons", http.StatusNotFound, nil)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-absence-reasons", http.StatusOK, data)
}

// @Summary Create Attendance Absence Reason
// @Tags Attendance Absence Reason API
// @Param dto body CreateDTO true "Attendance Absence Reason Data"
// @Security BearerAuth
// @Router /attendance-absence-reasons [post]
func (c *AttendanceAbsenceReasonController) Create(ctx *gin.Context) {
	var dto CreateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "attendance-absence-reasons", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.create(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "attendance-absence-reasons", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-absence-reasons", http.StatusCreated, data.ID)
}

// @Summary Update Attendance Absence Reason
// @Tags Attendance Absence Reason API
// @Param id path string true "Attendance Absence Reason ID"
// @Param dto body UpdateDTO true "Attendance Absence Reason Data"
// @Security BearerAuth
// @Router /attendance-absence-reasons/{id} [put]
func (c *AttendanceAbsenceReasonController) Update(ctx *gin.Context) {
	var dto UpdateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "attendance-absence-reasons", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.update(ctx, ctx.Param("id"), dto)
	if err != nil {
		helpers.RespondError(ctx, "attendance-absence-reasons", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondError(ctx, "attendance-absence-reasons", http.StatusNotFound, nil)
		return
	}
	helpers.RespondSuccess(ctx, "attendance-absence-reasons", http.StatusOK, data.ID)
}

// @Summary Archive Attendance Absence Reason
// @Tags Attendance Absence Reason API
// @Param id path string true "Attendance Absence Reason ID"
// @Security BearerAuth
// @Router /attendance-absence-reasons/{id}/archived [delete]
func (c *AttendanceAbsenceReasonController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "attendance-absence-reasons", func(id string) (bool, error) { return c.Service.archive(ctx, id) })
}

// @Summary Delete Attendance Absence Reason permanently
// @Tags Attendance Absence Reason API
// @Param id path string true "Attendance Absence Reason ID"
// @Security BearerAuth
// @Router /attendance-absence-reasons/{id} [delete]
func (c *AttendanceAbsenceReasonController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "attendance-absence-reasons", func(id string) (bool, error) { return c.Service.delete(ctx, id) })
}
