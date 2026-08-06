// Package presence provides a deliberately simple "online status": any
// authenticated request touches a short-TTL Redis key for that user (see
// middleware.TrackPresence), and IsOnline just checks whether the key is
// still alive. No heartbeats, no per-tab tracking, no WebSocket-specific
// path — regular API activity (which the frontend generates continuously
// via polling/queries) is a good enough proxy for "online" at this scale.
package presence

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const ttl = 2 * time.Minute

func Touch(ctx context.Context, r *redis.Client, userID string) {
	_ = r.Set(ctx, key(userID), "1", ttl).Err()
}

func IsOnline(ctx context.Context, r *redis.Client, userID string) bool {
	n, err := r.Exists(ctx, key(userID)).Result()
	return err == nil && n > 0
}

func key(userID string) string { return "presence:" + userID }
