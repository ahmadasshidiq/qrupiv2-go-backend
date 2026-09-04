package activities

import (
	"clasenna-go-backend/libs/helpers"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ActivityController struct{ Service *ActivityService }

func NewController(s *ActivityService) *ActivityController { return &ActivityController{Service: s} }

// @Summary      Get all Activities
// @Tags         Activity API
// @Param        sortBy     query  string  false  "Sort by field"
// @Param        sortOrder  query  string  false  "Sort order"  Enums(asc, desc)
// @Param        limit      query  int     false  "Number of results per page"
// @Param        page       query  int     false  "Page number"
// @Security     BearerAuth
// @Router       /activities [get]
func (c *ActivityController) GetAll(x *gin.Context) {
	var d DefaultFindDTO
	if e := x.ShouldBindQuery(&d); e != nil {
		helpers.RespondError(x, "activities", 400, e)
		return
	}
	v, e := c.Service.getAll(x, d)
	if e != nil {
		helpers.RespondError(x, "activities", 500, e)
		return
	}
	helpers.RespondSuccess(x, "activities", 200, v)
}

// @Summary      Get Activity by ID
// @Tags         Activity API
// @Param        id  path  string  true  "Activity ID"
// @Security     BearerAuth
// @Router       /activities/{id} [get]
func (c *ActivityController) GetByID(x *gin.Context) {
	v, e := c.Service.getByID(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, "activities", 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, "activities", 404, nil)
		return
	}
	helpers.RespondSuccess(x, "activities", 200, v)
}

// @Summary      Create Activity
// @Tags         Activity API
// @Param        dto  body  CreateDTO  true  "Activity Data"
// @Security     BearerAuth
// @Router       /activities [post]
func (c *ActivityController) Create(x *gin.Context) {
	var d CreateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, "activities", 400, e)
		return
	}
	v, e := c.Service.create(d)
	if e != nil {
		helpers.RespondError(x, "activities", 500, e)
		return
	}
	helpers.RespondSuccess(x, "activities", 201, v.ID)
}

// @Summary      Create Activities in bulk
// @Description  Creates up to 200 activities directly; larger requests up to 10,000 are queued and return a job ID
// @Tags         Activity API
// @Param        dto  body  BulkCreateDTO  true  "Bulk Activity Data"
// @Security     BearerAuth
// @Router       /activities/bulk [post]
func (c *ActivityController) CreateBulk(x *gin.Context) {
	var dto BulkCreateDTO
	if err := x.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(x, "activities", 400, err)
		return
	}
	if len(dto.Activities) > SynchronousBulkThreshold {
		result, err := c.Service.enqueueBulk(x.Request.Context(), dto, x.GetString("institution_id"))
		if err != nil {
			helpers.RespondError(x, "activities", http.StatusInternalServerError, err)
			return
		}
		helpers.RespondSuccess(x, "activities", http.StatusAccepted, result)
		return
	}
	result, err := c.Service.createBulk(dto)
	if err != nil {
		var validationError *BulkValidationError
		if errors.As(err, &validationError) {
			helpers.RespondErrorData(x, "activities", 400, validationError, err)
			return
		}
		helpers.RespondError(x, "activities", 500, err)
		return
	}
	helpers.RespondSuccess(x, "activities", 201, result)
}

// @Summary      Get Activity bulk job status
// @Tags         Activity API
// @Param        id  path  string  true  "Bulk Job ID"
// @Security     BearerAuth
// @Router       /activity-bulk-jobs/{id} [get]
func (c *ActivityController) GetBulkJob(x *gin.Context) {
	job, err := c.Service.getBulkJob(x.Param("id"), x.GetString("institution_id"))
	if err != nil {
		helpers.RespondError(x, "activity-bulk-jobs", http.StatusInternalServerError, err)
		return
	}
	if job == nil {
		helpers.RespondError(x, "activity-bulk-jobs", http.StatusNotFound, nil)
		return
	}
	helpers.RespondSuccess(x, "activity-bulk-jobs", http.StatusOK, job)
}

// @Summary      Update Activity
// @Tags         Activity API
// @Param        id   path  string     true  "Activity ID"
// @Param        dto  body  UpdateDTO  true  "Activity Data"
// @Security     BearerAuth
// @Router       /activities/{id} [put]
func (c *ActivityController) Update(x *gin.Context) {
	var d UpdateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, "activities", 400, e)
		return
	}
	v, e := c.Service.update(x.Param("id"), d)
	if e != nil {
		helpers.RespondError(x, "activities", 500, e)
		return
	}
	if v == nil {
		helpers.RespondError(x, "activities", 404, nil)
		return
	}
	helpers.RespondSuccess(x, "activities", 200, v.ID)
}

// @Summary      Archive Activity
// @Tags         Activity API
// @Param        id  path  string  true  "Activity ID"
// @Security     BearerAuth
// @Router       /activities/{id}/archived [delete]
func (c *ActivityController) Archive(x *gin.Context) {
	helpers.HandleRemove(x, "activities", c.Service.archive)
}

// @Summary      Delete Activity permanently
// @Tags         Activity API
// @Param        id  path  string  true  "Activity ID"
// @Security     BearerAuth
// @Router       /activities/{id} [delete]
func (c *ActivityController) Delete(x *gin.Context) {
	helpers.HandleRemove(x, "activities", c.Service.delete)
}
