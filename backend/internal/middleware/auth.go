package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/pkg/response"
)

// RequireAuth validates the JWT access token and stashes userID/role in
// c.Locals for downstream handlers and RequireRole.
func RequireAuth(tokens *auth.TokenService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			return response.Err(c, fiber.StatusUnauthorized, "missing bearer token")
		}

		claims, err := tokens.ParseAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			return response.Err(c, fiber.StatusUnauthorized, "invalid or expired token")
		}

		c.Locals("userID", claims.UserID)
		c.Locals("role", claims.Role)
		return c.Next()
	}
}

// RequireRole gates a route group to a set of org-level roles. Must run
// after RequireAuth.
func RequireRole(roles ...auth.Role) fiber.Handler {
	allowed := make(map[auth.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(auth.Role)
		if !allowed[role] {
			return response.Err(c, fiber.StatusForbidden, "insufficient permissions")
		}
		return c.Next()
	}
}
