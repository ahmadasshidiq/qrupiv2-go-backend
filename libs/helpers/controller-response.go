package helpers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RespondError writes the project's standard error response.
func RespondError(ctx *gin.Context, modelName string, statusCode int, err error) {
	RespondErrorData(ctx, modelName, statusCode, nil, err)
}

// RespondErrorData writes an error response with an optional descriptive payload.
func RespondErrorData(ctx *gin.Context, modelName string, statusCode int, data interface{}, err error) {
	ctx.JSON(statusCode, FormatResponse(ctx.Request.Method, modelName, statusCode, data, nil, err))
}

// RespondSuccess writes the project's standard success response.
func RespondSuccess(ctx *gin.Context, modelName string, statusCode int, data interface{}) {
	ctx.JSON(statusCode, FormatResponse(ctx.Request.Method, modelName, statusCode, data, nil, nil))
}

// HandleRemove handles the shared archive/permanent-delete controller flow.
func HandleRemove(ctx *gin.Context, modelName string, action func(string) (bool, error)) {
	found, err := action(ctx.Param("id"))
	if err != nil {
		RespondError(ctx, modelName, http.StatusInternalServerError, err)
		return
	}
	if !found {
		RespondError(ctx, modelName, http.StatusNotFound, nil)
		return
	}
	RespondSuccess(ctx, modelName, http.StatusOK, ctx.Param("id"))
}
