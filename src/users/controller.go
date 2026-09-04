package users

import (
	"clasenna-go-backend/libs/helpers"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	Service *UserService
}

func NewController(service *UserService) *UserController {
	return &UserController{Service: service}
}

// @Summary      Get all User
// @Tags         User API
// @Param        sortBy     query     string  false  "Sort by field"
// @Param        sortOrder  query     string  false  "Sort order (asc or desc)"  Enums(asc, desc)
// @Param        limit      query     int     false  "Number of results per page"
// @Param        page       query     int     false  "Page number"
// @Security     BearerAuth
// @Router       /users [get]
func (c *UserController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "users", http.StatusBadRequest, err)
		return
	}

	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "users", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "users", http.StatusNotFound, "User not found", nil)
		return
	}

	helpers.RespondSuccess(ctx, "users", http.StatusOK, data)
}

// @Summary      Get by id User
// @Tags         User API
// @Param        id  path  string  true  "User ID"
// @Security     BearerAuth
// @Router       /users/{id} [get]
func (c *UserController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")
	data, err := c.Service.getByID(ctx, id)
	if err != nil {
		helpers.RespondError(ctx, "users", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "users", http.StatusNotFound, "User not found", nil)
		return
	}

	helpers.RespondSuccess(ctx, "users", http.StatusOK, data)
}

// @Summary      Download student barcode
// @Tags         User API
// @Produce      image/png
// @Param        id  path  string  true  "Student user ID"
// @Success      200  {file}  binary
// @Security     BearerAuth
// @Router       /users/{id}/barcode [get]
func (c *UserController) GetBarcode(ctx *gin.Context) {
	data, err := c.Service.generateBarcodeImage(ctx, ctx.Param("id"))
	if err != nil {
		helpers.RespondError(ctx, "users", http.StatusNotFound, err)
		return
	}
	ctx.Header("Content-Disposition", "inline; filename=student-barcode.png")
	ctx.Data(http.StatusOK, "image/png", data)
}

// @Summary      Create User
// @Tags         User API
// @Param        dto  formData  CreateDTO  true  "User Data"
// @Param        file  formData  file  false  "Upload File (optional)"
// @Security     BearerAuth
// @Router       /users [post]
func (c *UserController) Create(ctx *gin.Context) {
	var dto CreateDTO

	if err := ctx.ShouldBind(&dto); err != nil {
		helpers.RespondError(ctx, "users", http.StatusBadRequest, err)
		return
	}

	data, err := c.Service.create(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "users", http.StatusInternalServerError, err)
		return
	}

	helpers.RespondSuccess(ctx, "users", http.StatusCreated, data.ID)
}

// @Summary      Update User
// @Tags         User API
// @Param        id    path      string     true  "User ID"
// @Param        dto   formData  UpdateDTO  true  "User Data"
// @Param        file  formData  file       false "Upload File (optional)"
// @Security     BearerAuth
// @Router       /users/{id} [put]
func (c *UserController) Update(ctx *gin.Context) {
	id := ctx.Param("id")

	var dto UpdateDTO
	if err := ctx.ShouldBind(&dto); err != nil {
		helpers.RespondError(ctx, "users", http.StatusBadRequest, err)
		return
	}

	data, err := c.Service.update(ctx, id, dto)
	if err != nil {
		helpers.RespondError(ctx, "users", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "users", http.StatusNotFound, "User not found", nil)
		return
	}

	helpers.RespondSuccess(ctx, "users", http.StatusOK, id)
}

// @Summary      Archive (soft delete) User
// @Tags         User API
// @Param        id  path  string  true  "User ID"
// @Security     BearerAuth
// @Router       /users/{id}/archived [delete]
func (c *UserController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "users", func(id string) (bool, error) {
		return c.Service.archive(ctx, id)
	})
}

// @Summary      Delete User permanently
// @Tags         User API
// @Param        id  path  string  true  "User ID"
// @Security     BearerAuth
// @Router       /users/{id} [delete]
func (c *UserController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "users", func(id string) (bool, error) {
		return c.Service.delete(ctx, id)
	})
}

// @Summary      Download Users Excel Template
// @Tags         User API
// @Router       /users/template-excel [get]
func (c *UserController) DownloadTemplateExcel(ctx *gin.Context) {
	data, err := c.Service.generateTemplateExcel()
	if err != nil {
		helpers.RespondError(ctx, "users", http.StatusInternalServerError, err)
		return
	}

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", "attachment; filename=users_template.xlsx")
	ctx.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// @Summary      Import Users by Excel
// @Tags         User API
// @Param        file  formData  file  true  "Excel file (.xlsx)"
// @Security     BearerAuth
// @Router       /users/import-excel [post]
func (c *UserController) ImportByExcel(ctx *gin.Context) {
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		helpers.RespondError(ctx, "users", http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	inserted, rowErrors, err := c.Service.importUsersByExcel(ctx, file)
	if err != nil {
		helpers.RespondError(ctx, "users", http.StatusInternalServerError, err)
		return
	}

	payload := gin.H{
		"inserted": inserted,
		"errors":   rowErrors,
		"message":  fmt.Sprintf("%d users imported", inserted),
	}

	helpers.RespondSuccess(ctx, "users", http.StatusOK, payload)
}
