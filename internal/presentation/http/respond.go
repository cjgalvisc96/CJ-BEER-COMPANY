// Package http is the presentation layer: Gin handlers that translate HTTP
// requests into application commands/queries and domain errors into status
// codes. It never touches domain types directly — only application DTOs.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	shared "github.com/cjgalvisc96/cj-beer-company/internal/shared/domain"
)

type errorResponse struct {
	Error string `json:"error"`
}

// respondError maps domain error kinds to HTTP status codes in one place,
// so use cases stay transport-agnostic.
func respondError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	if kind, ok := shared.KindOf(err); ok {
		message = err.Error()
		switch kind {
		case shared.KindValidation:
			status = http.StatusBadRequest
		case shared.KindNotFound:
			status = http.StatusNotFound
		case shared.KindConflict:
			status = http.StatusConflict
		case shared.KindUnprocessable:
			status = http.StatusUnprocessableEntity
		}
	} else {
		_ = c.Error(err)
	}
	c.JSON(status, errorResponse{Error: message})
}

func respondBindError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body: " + err.Error()})
}
