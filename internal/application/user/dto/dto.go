package dto

import (
	"time"

	"github.com/Oleja123/estate-agency/internal/domain/user"
	optional "github.com/denpa16/optional-go-type"
)

// DTOs parsed from JSON at the application boundary. These live in the
// application layer only.
type RegisterRequest struct {
	Email       string                  `json:"email"`
	Password    string                  `json:"password"`
	FirstName   string                  `json:"first_name"`
	LastName    string                  `json:"last_name"`
	PhoneNumber optional.OptionalString `json:"phone_number"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User         user.User `json:"user"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type UpdateProfileRequest struct {
	UserID      int                     `json:"-"`
	FirstName   optional.OptionalString `json:"first_name"`
	LastName    optional.OptionalString `json:"last_name"`
	PhoneNumber optional.OptionalString `json:"phone_number"`
	Email       optional.OptionalString `json:"email"`
	Role        optional.OptionalString `json:"role"`
}

type ChangePasswordRequest struct {
	UserID          int    `json:"-"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type ListUsersRequest struct {
	Filter user.Filter `json:"filter"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type ListUsersResponse struct {
	Users []user.User `json:"users"`
	Total int         `json:"total"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}
