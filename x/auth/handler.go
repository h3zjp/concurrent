// Package auth handles sever-side and client-side authentication
package auth

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"net/http"
)

var tracer = otel.Tracer("auth")

// Handler is the interface for handling HTTP requests
type Handler interface {
	GetPassport(c echo.Context) error
}

type handler struct {
	service Service
}

// NewHandler creates a new handler
func NewHandler(service Service) Handler {
	return &handler{service}
}

// Claim is used for get server signed jwt
// input user signed jwt
func (h *handler) GetPassport(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "HandlerGetPassport")
	defer span.End()

	remote := c.Param("remote")
	requester, ok := c.Get(RequesterIdCtxKey).(string)
	if !ok {
		return c.JSON(http.StatusForbidden, echo.Map{"status": "error", "message": "requester not found"})
	}

	response, err := h.service.IssuePassport(ctx, requester, remote)
	if err != nil {
		span.RecordError(err)
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"content": response})
}

