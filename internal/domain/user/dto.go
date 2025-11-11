package user

import (
	"time"

	"github.com/markphelps/optional"
)

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User         User      `json:"user"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type UpdateProfileRequest struct {
	UserID      int             `json:"-"`
	FirstName   optional.String `json:"first_name"`
	LastName    optional.String `json:"last_name"`
	PhoneNumber optional.String `json:"phone_number"`
	Email       optional.String `json:"email"`
}

type ChangePasswordRequest struct {
	UserID          int    `json:"-"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type UpdateUserRoleRequest struct {
	TargetUserID int  `json:"target_user_id"`
	RequesterID  int  `json:"-"`
	NewRole      Role `json:"new_role"`
}

type ListUsersRequest struct {
	Filter Filter `json:"filter"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type ListUsersResponse struct {
	Users []User `json:"users"`
	Total int    `json:"total"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}
