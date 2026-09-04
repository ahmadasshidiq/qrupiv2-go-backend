package activitycategories

import (
	"clasenna-go-backend/libs/helpers"
	"github.com/gin-gonic/gin"
)

type ActivityCategoryController struct{ Service *ActivityCategoryService }

func NewController(s *ActivityCategoryService) *ActivityCategoryController {
	return &ActivityCategoryController{Service: s}
}

// @Summary      Get all Activity Categories
// @Tags         Activity Category API
// @Param        sortBy     query  string  false  "Sort by field"
// @Param        sortOrder  query  string  false  "Sort order"  Enums(asc, desc)
// @Param        limit      query  int     false  "Number of results per page"
// @Param        page       query  int     false  "Page number"
// @Security     BearerAuth
// @Router       /activity-categories [get]
func (c *ActivityCategoryController) GetAll(x *gin.Context) {
	var d DefaultFindDTO
	if e := x.ShouldBindQuery(&d); e != nil {
		helpers.RespondError(x, "activity-categories", 400, e)
		return
	}
	v, e := c.Service.getAll(x, d)
	if e != nil {
		helpers.RespondError(x, "activity-categories", 500, e)
		return
	}
	helpers.RespondSuccess(x, "activity-categories", 200, v)
}

// @Summary      Get Activity Category by ID
// @Tags         Activity Category API
// @Param        id  path  string  true  "Activity Category ID"
// @Security     BearerAuth
// @Router       /activity-categories/{id} [get]
func (c *ActivityCategoryController) GetByID(x *gin.Context) {
	v, e := c.Service.getByID(x, x.Param("id"))
	if e != nil {
		helpers.RespondError(x, "activity-categories", 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, "activity-categories", 404, nil)
		return
	}
	helpers.RespondSuccess(x, "activity-categories", 200, v)
}

// @Summary      Create Activity Category
// @Tags         Activity Category API
// @Param        dto  body  CreateDTO  true  "Activity Category Data"
// @Security     BearerAuth
// @Router       /activity-categories [post]
func (c *ActivityCategoryController) Create(x *gin.Context) {
	var d CreateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, "activity-categories", 400, e)
		return
	}
	v, e := c.Service.create(x, d)
	if e != nil {
		helpers.RespondError(x, "activity-categories", 500, e)
		return
	}
	helpers.RespondSuccess(x, "activity-categories", 201, v.ID)
}

// @Summary      Update Activity Category
// @Tags         Activity Category API
// @Param        id   path  string     true  "Activity Category ID"
// @Param        dto  body  UpdateDTO  true  "Activity Category Data"
// @Security     BearerAuth
// @Router       /activity-categories/{id} [put]
func (c *ActivityCategoryController) Update(x *gin.Context) {
	var d UpdateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, "activity-categories", 400, e)
		return
	}
	v, e := c.Service.update(x, x.Param("id"), d)
	if e != nil {
		helpers.RespondError(x, "activity-categories", 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, "activity-categories", 404, nil)
		return
	}
	helpers.RespondSuccess(x, "activity-categories", 200, v.ID)
}

// @Summary      Archive Activity Category
// @Tags         Activity Category API
// @Param        id  path  string  true  "Activity Category ID"
// @Security     BearerAuth
// @Router       /activity-categories/{id}/archived [delete]
func (c *ActivityCategoryController) Archive(x *gin.Context) {
	helpers.HandleRemove(x, "activity-categories", func(id string) (bool, error) { return c.Service.archive(x, id) })
}

// @Summary      Delete Activity Category permanently
// @Tags         Activity Category API
// @Param        id  path  string  true  "Activity Category ID"
// @Security     BearerAuth
// @Router       /activity-categories/{id} [delete]
func (c *ActivityCategoryController) Delete(x *gin.Context) {
	helpers.HandleRemove(x, "activity-categories", func(id string) (bool, error) { return c.Service.delete(x, id) })
}
