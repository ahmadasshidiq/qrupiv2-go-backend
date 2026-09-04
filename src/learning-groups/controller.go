package learning_groups

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LearningGroupController struct{ Service *LearningGroupService }

func NewController(service *LearningGroupService) *LearningGroupController {
	return &LearningGroupController{Service: service}
}

// @Summary Get all Learning Group
// @Tags Learning Group API
// @Param sortBy query string false "Sort by field"
// @Param sortOrder query string false "Sort order" Enums(asc, desc)
// @Param limit query int false "Number of results per page"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /learning-groups [get]
func (c *LearningGroupController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "learning-groups", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "learning-groups", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "learning-groups", http.StatusOK, data)
}

// @Summary Get Learning Group by ID
// @Tags Learning Group API
// @Param id path string true "Learning Group ID"
// @Security BearerAuth
// @Router /learning-groups/{id} [get]
func (c *LearningGroupController) GetByID(ctx *gin.Context) {
	data, err := c.Service.getByID(ctx.Param("id"))
	if err != nil {
		helpers.RespondError(ctx, "learning-groups", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "learning-groups", http.StatusNotFound, "Learning Group not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "learning-groups", http.StatusOK, data)
}

// @Summary Create Learning Group
// @Tags Learning Group API
// @Param dto body CreateDTO true "Learning Group Data"
// @Security BearerAuth
// @Router /learning-groups [post]
func (c *LearningGroupController) Create(ctx *gin.Context) {
	var dto CreateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "learning-groups", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.create(dto)
	if err != nil {
		helpers.RespondError(ctx, "learning-groups", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "learning-groups", http.StatusCreated, data.ID)
}

// @Summary Update Learning Group
// @Tags Learning Group API
// @Param id path string true "Learning Group ID"
// @Param dto body UpdateDTO true "Learning Group Data"
// @Security BearerAuth
// @Router /learning-groups/{id} [put]
func (c *LearningGroupController) Update(ctx *gin.Context) {
	var dto UpdateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "learning-groups", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.update(ctx.Param("id"), dto)
	if err != nil {
		helpers.RespondError(ctx, "learning-groups", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "learning-groups", http.StatusNotFound, "Learning Group not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "learning-groups", http.StatusOK, data.ID)
}

// @Summary Archive Learning Group
// @Tags Learning Group API
// @Param id path string true "Learning Group ID"
// @Security BearerAuth
// @Router /learning-groups/{id}/archived [delete]
func (c *LearningGroupController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "learning-groups", c.Service.archive)
}

// @Summary Delete Learning Group permanently
// @Tags Learning Group API
// @Param id path string true "Learning Group ID"
// @Security BearerAuth
// @Router /learning-groups/{id} [delete]
func (c *LearningGroupController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "learning-groups", c.Service.delete)
}
