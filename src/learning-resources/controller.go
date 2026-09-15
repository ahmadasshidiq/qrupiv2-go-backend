package learning_resources

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LearningResourceController struct{ Service *LearningResourceService }

func NewController(service *LearningResourceService) *LearningResourceController {
	return &LearningResourceController{Service: service}
}

// @Summary Get all Learning Resource
// @Tags Learning Resource API
// @Param sortBy query string false "Sort by field"
// @Param sortOrder query string false "Sort order" Enums(asc, desc)
// @Param limit query int false "Number of results per page"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /learning-resources [get]
func (c *LearningResourceController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "learning-resources", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "learning-resources", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "learning-resources", http.StatusOK, data)
}

// @Summary Get Learning Resource by ID
// @Tags Learning Resource API
// @Param id path string true "Learning Resource ID"
// @Security BearerAuth
// @Router /learning-resources/{id} [get]
func (c *LearningResourceController) GetByID(ctx *gin.Context) {
	data, err := c.Service.getByID(ctx.Param("id"))
	if err != nil {
		helpers.RespondError(ctx, "learning-resources", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "learning-resources", http.StatusNotFound, "Learning Resource not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "learning-resources", http.StatusOK, data)
}

// @Summary Create Learning Resource
// @Tags Learning Resource API
// @Param dto formData CreateDTO true "Learning Resource Data"
// @Param file formData file false "Learning resource file"
// @Security BearerAuth
// @Router /learning-resources [post]
func (c *LearningResourceController) Create(ctx *gin.Context) {
	var dto CreateDTO
	if err := ctx.ShouldBind(&dto); err != nil {
		helpers.RespondError(ctx, "learning-resources", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.create(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "learning-resources", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "learning-resources", http.StatusCreated, data.ID)
}

// @Summary Update Learning Resource
// @Tags Learning Resource API
// @Param id path string true "Learning Resource ID"
// @Param dto formData UpdateDTO true "Learning Resource Data"
// @Param file formData file false "Replacement learning resource file"
// @Security BearerAuth
// @Router /learning-resources/{id} [put]
func (c *LearningResourceController) Update(ctx *gin.Context) {
	var dto UpdateDTO
	if err := ctx.ShouldBind(&dto); err != nil {
		helpers.RespondError(ctx, "learning-resources", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.update(ctx, ctx.Param("id"), dto)
	if err != nil {
		helpers.RespondError(ctx, "learning-resources", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "learning-resources", http.StatusNotFound, "Learning Resource not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "learning-resources", http.StatusOK, data.ID)
}

// @Summary Archive Learning Resource
// @Tags Learning Resource API
// @Param id path string true "Learning Resource ID"
// @Security BearerAuth
// @Router /learning-resources/{id}/archived [delete]
func (c *LearningResourceController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "learning-resources", c.Service.archive)
}

// @Summary Delete Learning Resource permanently
// @Tags Learning Resource API
// @Param id path string true "Learning Resource ID"
// @Security BearerAuth
// @Router /learning-resources/{id} [delete]
func (c *LearningResourceController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "learning-resources", c.Service.delete)
}
