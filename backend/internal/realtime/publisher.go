package realtime

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

// ProjectTopic and UserTopic are the two channel namespaces this app
// uses: broadcasts scoped to everyone viewing a project (Kanban, Gantt,
// ...) and broadcasts scoped to one person (notifications). Keeping topic
// construction here (not left to callers) means a project ID and a user
// ID can never collide on the same channel.
func ProjectTopic(projectID string) string { return "project:" + projectID }
func UserTopic(userID string) string       { return "user:" + userID }

// Publisher publishes to Redis. Any module can depend on it through a
// small local interface matching PublishToProject or PublishToUser —
// see task.EventPublisher — instead of importing this package directly.
type Publisher struct {
	redis *redis.Client
}

func NewPublisher(redisClient *redis.Client) *Publisher {
	return &Publisher{redis: redisClient}
}

func (p *Publisher) PublishToProject(ctx context.Context, projectID string, event any) error {
	return p.publish(ctx, ProjectTopic(projectID), event)
}

func (p *Publisher) PublishToUser(ctx context.Context, userID string, event any) error {
	return p.publish(ctx, UserTopic(userID), event)
}

func (p *Publisher) publish(ctx context.Context, topic string, event any) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.redis.Publish(ctx, topic, data).Err()
}

// RunSubscriber blocks, relaying every Redis pub/sub message to the
// matching topic's locally-connected WebSocket clients via hub. Run it in
// its own goroutine; it returns when ctx is cancelled. Subscribing to "*"
// is safe here because this Redis instance is dedicated to the app — no
// other pub/sub traffic shares it.
func RunSubscriber(ctx context.Context, redisClient *redis.Client, hub *Hub) {
	pubsub := redisClient.PSubscribe(ctx, "*")
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
			hub.BroadcastLocal(msg.Channel, []byte(msg.Payload))
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
