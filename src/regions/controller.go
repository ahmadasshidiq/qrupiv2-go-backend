package regions

import (
	"net/http"

	"clasenna-go-backend/libs/helpers"
	"github.com/gin-gonic/gin"
)

type RegionController struct{ Service *RegionService }

func NewController(service *RegionService) *RegionController {
	return &RegionController{Service: service}
}

// @Summary Get all provinces
// @Tags Region API
// @Param search query string false "Search province name"
// @Param sortBy query string false "Sort field" Enums(code,name,created_at,updated_at)
// @Param sortOrder query string false "Sort order" Enums(asc,desc)
// @Param limit query int false "Results per page (max 500)"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /provinces [get]
func (c *RegionController) GetProvinces(ctx *gin.Context) {
	c.respondList(ctx, "provinces", c.Service.getProvinces)
}

// @Summary Get province by code
// @Tags Region API
// @Param code path string true "Province code"
// @Security BearerAuth
// @Router /provinces/{code} [get]
func (c *RegionController) GetProvince(ctx *gin.Context) {
	data, err := c.Service.getProvince(ctx.Param("code"))
	respondOne(ctx, "provinces", data, err)
}

// @Summary Get all regencies
// @Tags Region API
// @Param province_code query string false "Filter by province code"
// @Param search query string false "Search regency name"
// @Param sortBy query string false "Sort field" Enums(code,name,created_at,updated_at)
// @Param sortOrder query string false "Sort order" Enums(asc,desc)
// @Param limit query int false "Results per page (max 500)"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /regencies [get]
func (c *RegionController) GetRegencies(ctx *gin.Context) {
	c.respondList(ctx, "regencies", c.Service.getRegencies)
}

// @Summary Get regency by code
// @Tags Region API
// @Param code path string true "Regency code"
// @Security BearerAuth
// @Router /regencies/{code} [get]
func (c *RegionController) GetRegency(ctx *gin.Context) {
	data, err := c.Service.getRegency(ctx.Param("code"))
	respondOne(ctx, "regencies", data, err)
}

// @Summary Get all districts
// @Tags Region API
// @Param regency_code query string false "Filter by regency code"
// @Param search query string false "Search district name"
// @Param sortBy query string false "Sort field" Enums(code,name,created_at,updated_at)
// @Param sortOrder query string false "Sort order" Enums(asc,desc)
// @Param limit query int false "Results per page (max 500)"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /districts [get]
func (c *RegionController) GetDistricts(ctx *gin.Context) {
	c.respondList(ctx, "districts", c.Service.getDistricts)
}

// @Summary Get district by code
// @Tags Region API
// @Param code path string true "District code"
// @Security BearerAuth
// @Router /districts/{code} [get]
func (c *RegionController) GetDistrict(ctx *gin.Context) {
	data, err := c.Service.getDistrict(ctx.Param("code"))
	respondOne(ctx, "districts", data, err)
}

// @Summary Get all villages
// @Tags Region API
// @Param district_code query string false "Filter by district code"
// @Param search query string false "Search village name"
// @Param sortBy query string false "Sort field" Enums(code,name,created_at,updated_at)
// @Param sortOrder query string false "Sort order" Enums(asc,desc)
// @Param limit query int false "Results per page (max 500)"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /villages [get]
func (c *RegionController) GetVillages(ctx *gin.Context) {
	c.respondList(ctx, "villages", c.Service.getVillages)
}

// @Summary Get village by code
// @Tags Region API
// @Param code path string true "Village code"
// @Security BearerAuth
// @Router /villages/{code} [get]
func (c *RegionController) GetVillage(ctx *gin.Context) {
	data, err := c.Service.getVillage(ctx.Param("code"))
	respondOne(ctx, "villages", data, err)
}

func (c *RegionController) respondList(ctx *gin.Context, model string, action func(FindDTO) (*helpers.PaginatedResult, error)) {
	var dto FindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, model, http.StatusBadRequest, err)
		return
	}
	data, err := action(dto)
	if err != nil {
		helpers.RespondError(ctx, model, http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, model, http.StatusOK, data)
}

func respondOne[T any](ctx *gin.Context, model string, data *T, err error) {
	if err != nil {
		helpers.RespondError(ctx, model, http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondError(ctx, model, http.StatusNotFound, nil)
		return
	}
	helpers.RespondSuccess(ctx, model, http.StatusOK, data)
}
