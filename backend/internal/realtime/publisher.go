package realtime

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

const channelPrefix = "project:"

func channelName(projectID string) string { return channelPrefix + projectID }

// Publisher publishes project-scoped events to Redis. Any module (task,
// kanban, comments, ...) can depend on this via a small local interface —
// see task.EventPublisher — instead of importing this package directly.
type Publisher struct {
	redis *redis.Client
}

func NewPublisher(redisClient *redis.Client) *Publisher {
	return &Publisher{redis: redisClient}
}

func (p *Publisher) Publish(ctx context.Context, projectID string, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.redis.Publish(ctx, channelName(projectID), data).Err()
}

// RunSubscriber blocks, relaying every "project:*" Redis message to the
// matching project's locally-connected WebSocket clients via hub. Run it
// in its own goroutine; it returns when ctx is cancelled.
func RunSubscriber(ctx context.Context, redisClient *redis.Client, hub *Hub) {
	pubsub := redisClient.PSubscribe(ctx, channelPrefix+"*")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			projectID := strings.TrimPrefix(msg.Channel, channelPrefix)
			hub.BroadcastLocal(projectID, []byte(msg.Payload))
		}
	}
}

// LogAndDrop is a convenience for call sites that publish best-effort and
// only want to log a failure rather than fail the request.
func LogAndDrop(err error) {
	if err != nil {
		log.Printf("realtime: publish failed: %v", err)
	}
}
