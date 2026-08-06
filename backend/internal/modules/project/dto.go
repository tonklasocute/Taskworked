package project

import "time"

type CreateRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=150"`
	Description string  `json:"description" validate:"max=2000"`
	DueDate     *string `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	Color       string  `json:"color" validate:"max=20"`
	Icon        string  `json:"icon" validate:"max=50"`
}

type UpdateRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=2,max=150"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	Status      *Status `json:"status" validate:"omitempty,oneof=planning active on_hold completed archived"`
	DueDate     *string `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	Color       *string `json:"color" validate:"omitempty,max=20"`
	Icon        *string `json:"icon" validate:"omitempty,max=50"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

type ListFilter struct {
	Status   Status
	Search   string
	Page     int
	PageSize int
}

type Response struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	OwnerID     string     `json:"owner_id"`
	Status      Status     `json:"status"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Color       string     `json:"color"`
	Icon        string     `json:"icon"`
	CreatedAt   time.Time  `json:"created_at"`
}

type ListResponse struct {
	Items    []Response `json:"items"`
	Total    int64      `json:"total"`
	Page     int        `json:"page"`
	PageSize int        `json:"page_size"`
}

func toResponse(p *Project) Response {
	return Response{
		ID:          p.ID.String(),
		Name:        p.Name,
		Description: p.Description,
		OwnerID:     p.OwnerID.String(),
		Status:      p.Status,
		DueDate:     p.DueDate,
		Color:       p.Color,
		Icon:        p.Icon,
		CreatedAt:   p.CreatedAt,
	}
}
