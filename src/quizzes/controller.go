package quizzes

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type QuizController struct{ Service *QuizService }

func NewController(service *QuizService) *QuizController {
	return &QuizController{Service: service}
}

// @Summary Get all Quiz
// @Tags Quiz API
// @Param sortBy query string false "Sort by field"
// @Param sortOrder query string false "Sort order" Enums(asc, desc)
// @Param limit query int false "Number of results per page"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /quizzes [get]
func (c *QuizController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "quizzes", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "quizzes", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "quizzes", http.StatusOK, data)
}

// @Summary Get Quiz by ID
// @Tags Quiz API
// @Param id path string true "Quiz ID"
// @Security BearerAuth
// @Router /quizzes/{id} [get]
func (c *QuizController) GetByID(ctx *gin.Context) {
	data, err := c.Service.getByID(ctx.Param("id"))
	if err != nil {
		helpers.RespondError(ctx, "quizzes", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "quizzes", http.StatusNotFound, "Quiz not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "quizzes", http.StatusOK, data)
}

// @Summary Create Quiz
// @Tags Quiz API
// @Param dto body CreateDTO true "Quiz Data"
// @Security BearerAuth
// @Router /quizzes [post]
func (c *QuizController) Create(ctx *gin.Context) {
	var dto CreateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "quizzes", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.create(dto)
	if err != nil {
		helpers.RespondError(ctx, "quizzes", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "quizzes", http.StatusCreated, data.ID)
}

// @Summary Update Quiz
// @Tags Quiz API
// @Param id path string true "Quiz ID"
// @Param dto body UpdateDTO true "Quiz Data"
// @Security BearerAuth
// @Router /quizzes/{id} [put]
func (c *QuizController) Update(ctx *gin.Context) {
	var dto UpdateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "quizzes", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.update(ctx.Param("id"), dto)
	if err != nil {
		helpers.RespondError(ctx, "quizzes", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "quizzes", http.StatusNotFound, "Quiz not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "quizzes", http.StatusOK, data.ID)
}

// @Summary Archive Quiz
// @Tags Quiz API
// @Param id path string true "Quiz ID"
// @Security BearerAuth
// @Router /quizzes/{id}/archived [delete]
func (c *QuizController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "quizzes", c.Service.archive)
}

// @Summary Delete Quiz permanently
// @Tags Quiz API
// @Param id path string true "Quiz ID"
// @Security BearerAuth
// @Router /quizzes/{id} [delete]
func (c *QuizController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "quizzes", c.Service.delete)
}
