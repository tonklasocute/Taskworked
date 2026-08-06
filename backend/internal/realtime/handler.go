package realtime

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/khomkrittk/taskworked/backend/internal/modules/auth"
	"github.com/khomkrittk/taskworked/backend/internal/modules/project"
)

// connAdapter makes *websocket.Conn satisfy Sender.
type connAdapter struct {
	conn *websocket.Conn
}

func (a *connAdapter) Send(data []byte) error {
	return a.conn.WriteMessage(websocket.TextMessage, data)
}

// RegisterRoute wires GET /ws?token=...&project_id=... — the browser
// WebSocket API can't set an Authorization header, so the access token
// travels as a query param instead, validated the same way as the
// Authorization header on every other route.
func RegisterRoute(router fiber.Router, tokens *auth.TokenService, projectSvc project.Service, hub *Hub) {
	router.Use("/ws", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}
		return c.Next()
	})

	router.Get("/ws", websocket.New(func(c *websocket.Conn) {
		handleConn(c, tokens, projectSvc, hub)
	}))
}

func handleConn(c *websocket.Conn, tokens *auth.TokenService, projectSvc project.Service, hub *Hub) {
	token := c.Query("token")
	projectID := c.Query("project_id")
	if token == "" || projectID == "" {
		closeWith(c, "missing token or project_id")
		return
	}

	claims, err := tokens.ParseAccessToken(token)
	if err != nil {
		closeWith(c, "invalid or expired token")
		return
	}

	actorID, err := uuid.Parse(claims.UserID)
	if err != nil {
		closeWith(c, "invalid user")
		return
	}
	projID, err := uuid.Parse(projectID)
	if err != nil {
		closeWith(c, "invalid project_id")
		return
	}

	isMember, _, err := projectSvc.CheckMembership(context.Background(), actorID, claims.Role, projID)
	if err != nil || !isMember {
		closeWith(c, "not a member of this project")
		return
	}

	sender := &connAdapter{conn: c}
	hub.Register(projectID, sender)
	defer hub.Unregister(projectID, sender)

	// Block on reads purely to detect disconnects; the client never sends
	// application messages, this is a server-to-client broadcast channel.
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			return
		}
	}
}

// closeWith sends a well-formed close frame. A close frame's payload must
// start with a 2-byte status code (RFC 6455 §5.5.1) — writing raw text via
// WriteMessage(CloseMessage, ...) produces a malformed frame that browsers
// and most WebSocket clients reject as "invalid status code" rather than
// reporting the reason, so FormatCloseMessage is required here.
func closeWith(c *websocket.Conn, reason string) {
	_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, reason))
}
