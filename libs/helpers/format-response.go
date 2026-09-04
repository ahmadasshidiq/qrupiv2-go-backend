package helpers

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
	Code    int    `json:"code,omitempty"`
}

func FormatResponse(method, modelName string, statusCode int, data interface{}, totalData *int64, err error) gin.H {
	// Capitalize only the first letter of the method
	method = strings.ToUpper(string(method[0])) + strings.ToLower(method[1:])

	if err != nil || statusCode < 200 || statusCode >= 300 {
		// Return standardized error response
		errorResp := gin.H{
			"status":  "error",
			"message": fmt.Sprintf("%s data %s unsuccessfully", method, modelName),
			"code":    statusCode,
		}

		if err != nil {
			errorResp["error"] = err.Error()
			if data != nil {
				errorResp["data"] = data
			}
		} else if data != nil {
			errorResp["error"] = fmt.Sprintf("%v", data)
		}

		return errorResp
	}

	// Success response
	resp := gin.H{
		"status":  "success",
		"message": fmt.Sprintf("%s data %s successfully", method, modelName),
		"data":    data,
	}

	if totalData != nil {
		resp["meta"] = gin.H{
			"totalData": *totalData,
		}
	}

	return resp
}
