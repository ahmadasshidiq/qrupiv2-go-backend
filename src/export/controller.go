package export

import (
	"clasenna-go-backend/libs/helpers"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ExportController struct {
	service *ExportService
}

func (c *ExportController) ExportQRS(ctx *gin.Context) {
	if ctx.GetString("institution_id") == "" {
		helpers.RespondError(ctx, "export", http.StatusForbidden, errors.New("institution scope is required"))
		return
	}
	path, err := c.service.generateQRS(ctx.GetString("institution_id"), ctx.Query("learning_group_id"))
	if err != nil {
		helpers.RespondError(ctx, "export", http.StatusBadRequest, err)
		return
	}
	ctx.FileAttachment(path, "student-qrs.pdf")
}

func NewExportController(service *ExportService) *ExportController {
	return &ExportController{service}
}

// @Summary      Export excel
// @Tags         Export API
// @Param        dto  body  ExportDTO  true  "Export Data"
// @Security     BearerAuth
// @Router       /export/excel [post]
func (c *ExportController) ExportExcel(ctx *gin.Context) {
	institutionID := ctx.GetString("institution_id")
	if institutionID == "" {
		res := helpers.FormatResponse(ctx.Request.Method, "export", http.StatusForbidden, nil, nil, errors.New("institution scope is required"))
		ctx.JSON(http.StatusForbidden, res)
		return
	}

	var dto ExportDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "export", http.StatusInternalServerError, nil, nil, err)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	if !allowedModels[dto.Models] {
		res := helpers.FormatResponse(ctx.Request.Method, "export", http.StatusBadRequest, nil, nil, errors.New("unsupported export model"))
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	if institutionScopedModels[dto.Models] {
		dto.Filters = append(dto.Filters, FilterDTO{Key: "institution_id", Value: institutionID})
	}
	if dto.Models == "users" || dto.Models == "learning_resources" || dto.Models == "attendance_logs" || dto.Models == "activities" || dto.Models == "quiz_sessions" {
		dto.Filters = append(dto.Filters, FilterDTO{Key: "deleted_at", Operator: "is null"})
	}

	data, err := c.service.exportToMap(dto, dto.Filters)
	if err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "export", http.StatusInternalServerError, nil, nil, err)
		ctx.JSON(http.StatusInternalServerError, res)
		return
	}

	filePath, err := c.service.generateExcel(dto, data)
	if err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "export", http.StatusBadRequest, nil, nil, err)
		ctx.JSON(http.StatusInternalServerError, res)
		return
	}

	ctx.FileAttachment(filePath, dto.Filename)
}
