package learning_group_members

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LearningGroupMemberController struct{ Service *LearningGroupMemberService }

func NewController(service *LearningGroupMemberService) *LearningGroupMemberController {
	return &LearningGroupMemberController{Service: service}
}

// @Summary Get all Learning Group Member
// @Tags Learning Group Member API
// @Param sortBy query string false "Sort by field"
// @Param sortOrder query string false "Sort order" Enums(asc, desc)
// @Param limit query int false "Number of results per page"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /learning-group-members [get]
func (c *LearningGroupMemberController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "learning-group-members", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "learning-group-members", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "learning-group-members", http.StatusOK, data)
}

// @Summary Get Learning Group Member by ID
// @Tags Learning Group Member API
// @Param id path string true "Learning Group Member ID"
// @Security BearerAuth
// @Router /learning-group-members/{id} [get]
func (c *LearningGroupMemberController) GetByID(ctx *gin.Context) {
	data, err := c.Service.getByID(ctx.Param("id"))
	if err != nil {
		helpers.RespondError(ctx, "learning-group-members", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "learning-group-members", http.StatusNotFound, "Learning Group Member not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "learning-group-members", http.StatusOK, data)
}

// @Summary Create Learning Group Member
// @Tags Learning Group Member API
// @Param dto body CreateDTO true "Learning Group Member Data"
// @Security BearerAuth
// @Router /learning-group-members [post]
func (c *LearningGroupMemberController) Create(ctx *gin.Context) {
	var dto CreateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "learning-group-members", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.create(dto)
	if err != nil {
		helpers.RespondError(ctx, "learning-group-members", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "learning-group-members", http.StatusCreated, data.ID)
}

// @Summary Update Learning Group Member
// @Tags Learning Group Member API
// @Param id path string true "Learning Group Member ID"
// @Param dto body UpdateDTO true "Learning Group Member Data"
// @Security BearerAuth
// @Router /learning-group-members/{id} [put]
func (c *LearningGroupMemberController) Update(ctx *gin.Context) {
	var dto UpdateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "learning-group-members", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.update(ctx.Param("id"), dto)
	if err != nil {
		helpers.RespondError(ctx, "learning-group-members", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "learning-group-members", http.StatusNotFound, "Learning Group Member not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "learning-group-members", http.StatusOK, data.ID)
}

// @Summary Archive Learning Group Member
// @Tags Learning Group Member API
// @Param id path string true "Learning Group Member ID"
// @Security BearerAuth
// @Router /learning-group-members/{id}/archived [delete]
func (c *LearningGroupMemberController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "learning-group-members", c.Service.archive)
}

// @Summary Delete Learning Group Member permanently
// @Tags Learning Group Member API
// @Param id path string true "Learning Group Member ID"
// @Security BearerAuth
// @Router /learning-group-members/{id} [delete]
func (c *LearningGroupMemberController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "learning-group-members", c.Service.delete)
}
