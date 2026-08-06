// Package httpctx holds request-context helpers shared by every module's
// HTTP handler: pulling the authenticated actor out of Fiber locals (set by
// middleware.RequireAuth) and mapping service errors to HTTP responses.
package httpctx

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	apperrors "github.com/khomkrittk/taskworked/backend/internal/pkg/errors"
	"github.com/khomkrittk/taskworked/backend/internal/pkg/response"
)

func ActorID(c *fiber.Ctx) (uuid.UUID, error) {
	raw, _ := c.Locals("userID").(string)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.Unauthorized("invalid user context")
	}
	return id, nil
}

// ActorRole returns the org-level role stashed by middleware.RequireAuth,
// as a plain string. This package stays independent of the auth module (to
// avoid an import cycle, since auth's own handler uses WriteErr below) —
// callers cast to their module's own role type, e.g. auth.Role(...).
func ActorRole(c *fiber.Ctx) string {
	role, _ := c.Locals("role").(string)
	return role
}

func WriteErr(c *fiber.Ctx, err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return response.Err(c, appErr.Status, appErr.Message)
	}
	return response.Err(c, fiber.StatusInternalServerError, "internal server error")
}
