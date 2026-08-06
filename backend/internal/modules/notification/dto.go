package notification

import "time"

type Response struct {
	ID        string    `json:"id"`
	Type      Type      `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Link      string    `json:"link,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type ListResponse struct {
	Items       []Response `json:"items"`
	UnreadCount int64      `json:"unread_count"`
}

type PreferenceResponse struct {
	EmailEnabled    bool   `json:"email_enabled"`
	LineEnabled     bool   `json:"line_enabled"`
	LineNotifyToken string `json:"line_notify_token,omitempty"`
}

type UpdatePreferenceRequest struct {
	EmailEnabled    *bool   `json:"email_enabled"`
	LineEnabled     *bool   `json:"line_enabled"`
	LineNotifyToken *string `json:"line_notify_token" validate:"omitempty,max=200"`
}

func toResponse(n *Notification) Response {
	return Response{
		ID:        n.ID.String(),
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Link:      n.Link,
		Read:      n.Read,
		CreatedAt: n.CreatedAt,
	}
}

func toPreferenceResponse(p *Preference) PreferenceResponse {
	return PreferenceResponse{
		EmailEnabled:    p.EmailEnabled,
		LineEnabled:     p.LineEnabled,
		LineNotifyToken: p.LineNotifyToken,
	}
}
