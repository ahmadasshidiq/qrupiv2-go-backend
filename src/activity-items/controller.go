package activityitems

import (
	"clasenna-go-backend/libs/helpers"

	"github.com/gin-gonic/gin"
)

type ActivityItemController struct{ Service *ActivityItemService }

func NewController(s *ActivityItemService) *ActivityItemController {
	return &ActivityItemController{Service: s}
}

// @Summary      Get all Activity Items
// @Tags         Activity Item API
// @Param        sortBy     query  string  false  "Sort by field"
// @Param        sortOrder  query  string  false  "Sort order"  Enums(asc, desc)
// @Param        limit      query  int     false  "Number of results per page"
// @Param        page       query  int     false  "Page number"
// @Security     BearerAuth
// @Router       /activity-items [get]
func (c *ActivityItemController) GetAll(x *gin.Context) {
	var d DefaultFindDTO
	if e := x.ShouldBindQuery(&d); e != nil {
		helpers.RespondError(x, "activity-items", 400, e)
		return
	}
	v, e := c.Service.getAll(x, d)
	if e != nil {
		helpers.RespondError(x, "activity-items", 500, e)
		return
	}
	helpers.RespondSuccess(x, "activity-items", 200, v)
}

// @Summary      Get Activity Item by ID
// @Tags         Activity Item API
// @Param        id  path  string  true  "Activity Item ID"
// @Security     BearerAuth
// @Router       /activity-items/{id} [get]
func (c *ActivityItemController) GetByID(x *gin.Context) {
	v, e := c.Service.getByID(x, x.Param("id"))
	if e != nil {
		helpers.RespondError(x, "activity-items", 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, "activity-items", 404, nil)
		return
	}
	helpers.RespondSuccess(x, "activity-items", 200, v)
}

// @Summary      Create Activity Item
// @Tags         Activity Item API
// @Param        dto  body  CreateDTO  true  "Activity Item Data"
// @Security     BearerAuth
// @Router       /activity-items [post]
func (c *ActivityItemController) Create(x *gin.Context) {
	var d CreateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, "activity-items", 400, e)
		return
	}
	v, e := c.Service.create(x, d)
	if e != nil {
		helpers.RespondError(x, "activity-items", 500, e)
		return
	}
	helpers.RespondSuccess(x, "activity-items", 201, v.ID)
}

// @Summary      Update Activity Item
// @Tags         Activity Item API
// @Param        id   path  string     true  "Activity Item ID"
// @Param        dto  body  UpdateDTO  true  "Activity Item Data"
// @Security     BearerAuth
// @Router       /activity-items/{id} [put]
func (c *ActivityItemController) Update(x *gin.Context) {
	var d UpdateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, "activity-items", 400, e)
		return
	}
	v, e := c.Service.update(x, x.Param("id"), d)
	if e != nil {
		helpers.RespondError(x, "activity-items", 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, "activity-items", 404, nil)
		return
	}
	helpers.RespondSuccess(x, "activity-items", 200, v.ID)
}

// @Summary      Archive Activity Item
// @Tags         Activity Item API
// @Param        id  path  string  true  "Activity Item ID"
// @Security     BearerAuth
// @Router       /activity-items/{id}/archived [delete]
func (c *ActivityItemController) Archive(x *gin.Context) {
	helpers.HandleRemove(x, "activity-items", func(id string) (bool, error) { return c.Service.archive(x, id) })
}

// @Summary      Delete Activity Item permanently
// @Tags         Activity Item API
// @Param        id  path  string  true  "Activity Item ID"
// @Security     BearerAuth
// @Router       /activity-items/{id} [delete]
func (c *ActivityItemController) Delete(x *gin.Context) {
	helpers.HandleRemove(x, "activity-items", func(id string) (bool, error) { return c.Service.delete(x, id) })
}
