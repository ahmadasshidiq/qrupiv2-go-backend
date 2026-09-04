package export

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ExportController struct {
	service *ExportService
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
	var dto ExportDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		res := helpers.FormatResponse(ctx.Request.Method, "export", http.StatusInternalServerError, nil, nil, err)
		ctx.JSON(http.StatusBadRequest, res)
		return
	}

	data, err := c.service.exportToMap(dto)
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
