package roles

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RoleController struct {
	Service *RoleService
}

func NewController(service *RoleService) *RoleController {
	return &RoleController{Service: service}
}

// @Summary      Get all Master Permissions
// @Tags         Role API
// @Router       /roles/master-permissions [get]
func (c *RoleController) GetMasterPermissions(ctx *gin.Context) {
	data, err := c.Service.getMasterPermissions()
	if err != nil {
		helpers.RespondError(ctx, "roles", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "roles", http.StatusOK, data)
}

// @Summary      Get all Role
// @Tags         Role API
// @Param        sortBy     query     string  false  "Sort by field"
// @Param        sortOrder  query     string  false  "Sort order (asc or desc)"  Enums(asc, desc)
// @Param        limit      query     int     false  "Number of results per page"
// @Param        page       query     int     false  "Page number"
// @Security     BearerAuth
// @Router       /roles [get]
func (c *RoleController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "roles", http.StatusBadRequest, err)
		return
	}

	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "roles", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "roles", http.StatusNotFound, "Role not found", nil)
		return
	}

	helpers.RespondSuccess(ctx, "roles", http.StatusOK, data)
}

// @Summary      Get by id Role
// @Tags         Role API
// @Param        id  path  string  true  "Role ID"
// @Security     BearerAuth
// @Router       /roles/{id} [get]
func (c *RoleController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")
	data, err := c.Service.getByID(id)
	if err != nil {
		helpers.RespondError(ctx, "roles", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "roles", http.StatusNotFound, "Role not found", nil)
		return
	}

	helpers.RespondSuccess(ctx, "roles", http.StatusOK, data)
}

// @Summary      Create Role
// @Tags         Role API
// @Param        dto  body  CreateDTO  true  "Role Data"
// @Security     BearerAuth
// @Router       /roles [post]
func (c *RoleController) Create(ctx *gin.Context) {
	var dto CreateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "roles", http.StatusBadRequest, err)
		return
	}

	data, err := c.Service.create(dto)
	if err != nil {
		helpers.RespondError(ctx, "roles", http.StatusInternalServerError, err)
		return
	}

	helpers.RespondSuccess(ctx, "roles", http.StatusCreated, data.ID)
}

// @Summary      Update Role
// @Tags         Role API
// @Param        id   path  string         true  "Role ID"
// @Param        dto  body  UpdateDTO  true  "Role Data"
// @Security     BearerAuth
// @Router       /roles/{id} [put]
func (c *RoleController) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var dto UpdateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "roles", http.StatusBadRequest, err)
		return
	}

	data, err := c.Service.update(id, dto)
	if err != nil {
		helpers.RespondError(ctx, "roles", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "roles", http.StatusNotFound, "Role not found", nil)
		return
	}

	helpers.RespondSuccess(ctx, "roles", http.StatusOK, id)
}

// @Summary      Archive (soft delete) Role
// @Tags         Role API
// @Param        id  path  string  true  "Role ID"
// @Security     BearerAuth
// @Router       /roles/{id}/archived [delete]
func (c *RoleController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "roles", c.Service.archive)
}

// @Summary      Delete Role permanently
// @Tags         Role API
// @Param        id  path  string  true  "Role ID"
// @Security     BearerAuth
// @Router       /roles/{id} [delete]
func (c *RoleController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "roles", c.Service.delete)
}
