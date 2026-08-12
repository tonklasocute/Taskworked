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

// ActorOrgID returns the organization_id claim stashed by
// middleware.RequireAuth. This is the stateless/cheap read path (see
// auth.Claims.OrganizationID) — most authorization code should prefer a
// fresh server-side lookup (auth.Service.GetOrganizationID) over trusting
// this claim, so a user removed from an organization loses access
// immediately rather than only after their token expires. Use this helper
// where a DB round-trip isn't otherwise happening (e.g. WebSocket handshake,
// structured logging), not as the enforcement mechanism itself.
func ActorOrgID(c *fiber.Ctx) (uuid.UUID, error) {
	raw, _ := c.Locals("organizationID").(string)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperrors.Unauthorized("invalid organization context")
	}
	return id, nil
}

func WriteErr(c *fiber.Ctx, err error) error {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return response.Err(c, appErr.Status, appErr.Message)
	}
	return response.Err(c, fiber.StatusInternalServerError, "internal server error")
}
