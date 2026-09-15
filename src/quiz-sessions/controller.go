package quiz_sessions

import (
	"clasenna-go-backend/libs/helpers"
	"net/http"

	"github.com/gin-gonic/gin"
)

type QuizSessionController struct{ Service *QuizSessionService }

func NewController(service *QuizSessionService) *QuizSessionController {
	return &QuizSessionController{Service: service}
}

// @Summary Get all Quiz Session
// @Tags Quiz Session API
// @Param sortBy query string false "Sort by field"
// @Param sortOrder query string false "Sort order" Enums(asc, desc)
// @Param limit query int false "Number of results per page"
// @Param page query int false "Page number"
// @Security BearerAuth
// @Router /quiz-sessions [get]
func (c *QuizSessionController) GetAll(ctx *gin.Context) {
	var dto DefaultFindDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "quiz-sessions", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.getAll(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "quiz-sessions", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "quiz-sessions", http.StatusOK, data)
}

// @Summary Get student quiz rankings
// @Description Returns school, class, or learning-group rankings using completed quiz-session scores
// @Tags Quiz Session API
// @Param scope query string true "Ranking scope" Enums(school, class, learning_group)
// @Param learning_group_id query string false "Required for class and learning_group scope"
// @Param quiz_id query string false "Quiz ID"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Param limit query int false "Ranking limit (default 50, maximum 100)"
// @Security BearerAuth
// @Router /quiz-sessions/rankings [get]
func (c *QuizSessionController) GetRankings(ctx *gin.Context) {
	var dto RankingFilterDTO
	if err := ctx.ShouldBindQuery(&dto); err != nil {
		helpers.RespondError(ctx, "quiz-sessions", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.getRankings(ctx, dto)
	if err != nil {
		helpers.RespondError(ctx, "quiz-sessions", http.StatusBadRequest, err)
		return
	}
	helpers.RespondSuccess(ctx, "quiz-sessions", http.StatusOK, data)
}

// @Summary Get Quiz Session by ID
// @Tags Quiz Session API
// @Param id path string true "Quiz Session ID"
// @Security BearerAuth
// @Router /quiz-sessions/{id} [get]
func (c *QuizSessionController) GetByID(ctx *gin.Context) {
	data, err := c.Service.getByID(ctx.Param("id"))
	if err != nil {
		helpers.RespondError(ctx, "quiz-sessions", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "quiz-sessions", http.StatusNotFound, "Quiz Session not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "quiz-sessions", http.StatusOK, data)
}

// @Summary Create Quiz Session
// @Tags Quiz Session API
// @Param dto body CreateDTO true "Quiz Session Data"
// @Security BearerAuth
// @Router /quiz-sessions [post]
func (c *QuizSessionController) Create(ctx *gin.Context) {
	var dto CreateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "quiz-sessions", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.create(dto)
	if err != nil {
		helpers.RespondError(ctx, "quiz-sessions", http.StatusInternalServerError, err)
		return
	}
	helpers.RespondSuccess(ctx, "quiz-sessions", http.StatusCreated, data.ID)
}

// @Summary Update Quiz Session
// @Tags Quiz Session API
// @Param id path string true "Quiz Session ID"
// @Param dto body UpdateDTO true "Quiz Session Data"
// @Security BearerAuth
// @Router /quiz-sessions/{id} [put]
func (c *QuizSessionController) Update(ctx *gin.Context) {
	var dto UpdateDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		helpers.RespondError(ctx, "quiz-sessions", http.StatusBadRequest, err)
		return
	}
	data, err := c.Service.update(ctx.Param("id"), dto)
	if err != nil {
		helpers.RespondError(ctx, "quiz-sessions", http.StatusInternalServerError, err)
		return
	}
	if data == nil {
		helpers.RespondErrorData(ctx, "quiz-sessions", http.StatusNotFound, "Quiz Session not found", nil)
		return
	}
	helpers.RespondSuccess(ctx, "quiz-sessions", http.StatusOK, data.ID)
}

// @Summary Archive Quiz Session
// @Tags Quiz Session API
// @Param id path string true "Quiz Session ID"
// @Security BearerAuth
// @Router /quiz-sessions/{id}/archived [delete]
func (c *QuizSessionController) Archive(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "quiz-sessions", c.Service.archive)
}

// @Summary Delete Quiz Session permanently
// @Tags Quiz Session API
// @Param id path string true "Quiz Session ID"
// @Security BearerAuth
// @Router /quiz-sessions/{id} [delete]
func (c *QuizSessionController) Delete(ctx *gin.Context) {
	helpers.HandleRemove(ctx, "quiz-sessions", c.Service.delete)
}
