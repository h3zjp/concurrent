// Package auth handles sever-side and client-side authentication
package auth

import (
	"github.com/labstack/echo/v4"
	"github.com/totegamma/concurrent/x/core"
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

	requester, ok := c.Get(core.RequesterIdCtxKey).(string)
	if !ok {
		return c.JSON(http.StatusForbidden, echo.Map{"status": "error", "message": "requester not found"})
	}

	keys, ok := c.Get(core.RequesterKeychainKey).([]core.Key)

	response, err := h.service.IssuePassport(ctx, requester, keys)
	if err != nil {
		span.RecordError(err)
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"content": response})
}
