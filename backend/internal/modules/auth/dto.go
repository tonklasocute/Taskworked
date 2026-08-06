package auth

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Role         Role    `json:"role"`
	AvatarURL    string  `json:"avatar_url"`
	DepartmentID *string `json:"department_id,omitempty"`
}

type UpdateRoleRequest struct {
	Role Role `json:"role" validate:"required,oneof=super_admin admin manager leader employee"`
}

type UpdateDepartmentRequest struct {
	DepartmentID *string `json:"department_id" validate:"omitempty,uuid"`
}

func toUserResponse(u *User) UserResponse {
	var departmentID *string
	if u.DepartmentID != nil {
		s := u.DepartmentID.String()
		departmentID = &s
	}
	return UserResponse{
		ID:           u.ID.String(),
		Name:         u.Name,
		Email:        u.Email,
		Role:         u.Role,
		AvatarURL:    u.AvatarURL,
		DepartmentID: departmentID,
	}
}
