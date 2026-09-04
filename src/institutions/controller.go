package institutions

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type InstitutionController struct {
	Service *InstitutionService
}

func NewController(service *InstitutionService) *InstitutionController {
	return &InstitutionController{Service: service}
}

// @Summary      Get all Institution
// @Tags         Institution API
// @Param        sortBy     query     string  false  "Sort by field"
// @Param        sortOrder  query     string  false  "Sort order (asc or desc)"  Enums(asc, desc)
// @Param        limit      query     int     false  "Number of results per page"
// @Param        page       query     int     false  "Page number"
// @Security     BearerAuth
// @Router       /institutions [get]
func (c *InstitutionController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "institutions", http.StatusBadRequest, err)
		return
	}

	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "institutions", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "institutions", http.StatusNotFound, "Institution not found", nil)
		return
	}

	helpers.RespondSuccess(ctx, "institutions", http.StatusOK, data)
}

// @Summary      Get by id Institution
// @Tags         Institution API
// @Param        id  path  string  true  "Institution ID"
// @Security     BearerAuth
// @Router       /institutions/{id} [get]
func (c *InstitutionController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")
	data, err := c.Service.getByID(id)
	if err != nil {
		helpers.RespondError(ctx, "institutions", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "institutions", http.StatusNotFound, "Institution not found", nil)
		return
	}

	helpers.RespondSuccess(ctx, "institutions", http.StatusOK, data)
}

// @Summary      Create Institution
// @Tags         Institution API
// @Param        dto  formData  CreateDTO  true  "Institution Data"
// @Param        file  formData  file  false  "Upload File (optional)"
// @Security     BearerAuth
// @Router       /institutions [post]
func (c *InstitutionController) Create(ctx *gin.Context) {
	var dto CreateDTO

	if err := ctx.ShouldBind(&dto); err != nil {
		helpers.RespondError(ctx, "institutions", http.StatusBadRequest, err)
		return
	}

	data, err := c.Service.create(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "institutions", http.StatusInternalServerError, err)
		return
	}

	helpers.RespondSuccess(ctx, "institutions", http.StatusCreated, data)
}

// @Summary      Update Institution
// @Tags         Institution API
// @Param        id    path      string     true  "Institution ID"
// @Param        dto   formData  UpdateDTO  true  "Institution Data"
// @Param        file  formData  file       false "Upload File (optional)"
// @Security     BearerAuth
// @Router       /institutions/{id} [put]
func (c *InstitutionController) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var dto UpdateDTO
	if err := ctx.ShouldBind(&dto); err != nil {
		helpers.RespondError(ctx, "institutions", http.StatusBadRequest, err)
		return
	}

	data, err := c.Service.update(ctx, id, dto)
	if err != nil {
		helpers.RespondError(ctx, "institutions", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "institutions", http.StatusNotFound, "Institution not found", nil)
		return
	}

	helpers.RespondSuccess(ctx, "institutions", http.StatusOK, id)
}

// @Summary      Archive (soft delete) Institution
// @Tags         Institution API
// @Param        id  path  string  true  "Institution ID"
// @Security     BearerAuth
// @Router       /institutions/{id}/archived [delete]
func (c *InstitutionController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "institutions", c.Service.archive)
}

// @Summary      Delete Institution permanently
// @Tags         Institution API
// @Param        id  path  string  true  "Institution ID"
// @Security     BearerAuth
// @Router       /institutions/{id} [delete]
func (c *InstitutionController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "institutions", c.Service.delete)
}
