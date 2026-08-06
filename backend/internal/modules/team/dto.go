package team

import "time"

type CreateDepartmentRequest struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
}

type DepartmentResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// DirectoryEntry is one row of the Team page: a user enriched with their
// department name, online status, and current workload.
type DirectoryEntry struct {
	UserID         string `json:"user_id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	Role           string `json:"role"`
	AvatarURL      string `json:"avatar_url"`
	DepartmentID   string `json:"department_id,omitempty"`
	DepartmentName string `json:"department_name,omitempty"`
	Online         bool   `json:"online"`
	Workload       int64  `json:"workload"`
}

type DirectoryResponse struct {
	Members     []DirectoryEntry     `json:"members"`
	Departments []DepartmentResponse `json:"departments"`
}

func toDepartmentResponse(d *Department) DepartmentResponse {
	return DepartmentResponse{ID: d.ID.String(), Name: d.Name, CreatedAt: d.CreatedAt}
}
