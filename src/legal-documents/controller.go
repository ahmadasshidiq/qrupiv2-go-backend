package legal_documents

import (
	"clasenna-go-backend/libs/helpers"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Controller struct{ Service *Service }

func NewController(s *Service) *Controller { return &Controller{Service: s} }
func (c *Controller) GetAll(x *gin.Context) {
	var d DefaultFindDTO
	if e := x.ShouldBindQuery(&d); e != nil {
		helpers.RespondError(x, "legal-documents", 400, e)
		return
	}
	v, e := c.Service.getAll(x, d)
	if e != nil {
		helpers.RespondError(x, "legal-documents", 500, e)
		return
	}
	helpers.RespondSuccess(x, "legal-documents", 200, v)
}
func (c *Controller) GetByID(x *gin.Context) {
	v, e := c.Service.getByID(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, "legal-documents", 500, e)
		return
	}
	if v == nil {
		helpers.RespondErrorData(x, "legal-documents", 404, "Legal document not found", nil)
		return
	}
	helpers.RespondSuccess(x, "legal-documents", 200, v)
}
func (c *Controller) Create(x *gin.Context) {
	var d CreateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, "legal-documents", 400, e)
		return
	}
	v, e := c.Service.create(d)
	if e != nil {
		helpers.RespondError(x, "legal-documents", 500, e)
		return
	}
	helpers.RespondSuccess(x, "legal-documents", http.StatusCreated, v)
}
func (c *Controller) Update(x *gin.Context) {
	var d UpdateDTO
	if e := x.ShouldBindJSON(&d); e != nil {
		helpers.RespondError(x, "legal-documents", 400, e)
		return
	}
	v, e := c.Service.update(x.Param("id"), d)
	if e != nil {
		helpers.RespondError(x, "legal-documents", 500, e)
		return
	}
	if v == nil {
		helpers.RespondErrorData(x, "legal-documents", 404, "Legal document not found", nil)
		return
	}
	helpers.RespondSuccess(x, "legal-documents", 200, v)
}
func (c *Controller) Delete(x *gin.Context) {
	deleted, e := c.Service.delete(x.Param("id"))
	if e != nil {
		helpers.RespondError(x, "legal-documents", 500, e)
		return
	}
	if !deleted {
		helpers.RespondErrorData(x, "legal-documents", 404, "Legal document not found", nil)
		return
	}
	helpers.RespondSuccess(x, "legal-documents", 204, nil)
}
