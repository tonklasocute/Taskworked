package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/khomkrittk/taskworked/backend/internal/presence"
	"github.com/redis/go-redis/v9"
)

// TrackPresence marks the authenticated caller as online. Must run after
// RequireAuth (reads the userID it stashes in c.Locals).
func TrackPresence(redisClient *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if userID, ok := c.Locals("userID").(string); ok && userID != "" {
			presence.Touch(c.Context(), redisClient, userID)
		}
		return c.Next()
	}
}
