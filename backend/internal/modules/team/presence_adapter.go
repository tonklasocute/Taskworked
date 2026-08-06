package team

import (
	"context"

	"github.com/khomkrittk/taskworked/backend/internal/presence"
	"github.com/redis/go-redis/v9"
)

// RedisPresence adapts the presence package's free functions to the
// PresenceChecker interface, so service.go doesn't depend on Redis
// directly and can be tested with a fake instead.
type RedisPresence struct {
	Client *redis.Client
}

func (p *RedisPresence) IsOnline(ctx context.Context, userID string) bool {
	return presence.IsOnline(ctx, p.Client, userID)
}
