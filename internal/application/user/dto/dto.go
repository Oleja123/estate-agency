package dto

import (
	"time"

	"github.com/Oleja123/estate-agency/internal/domain/user"
	optional "github.com/denpa16/optional-go-type"
)

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
	User         PublicUser `json:"user"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	ExpiresAt    time.Time  `json:"expires_at"`
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
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type ListUsersRequest struct {
	Filter user.Filter `json:"filter"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type ListUsersResponse struct {
	Users []PublicUser `json:"users"`
	Total int          `json:"total"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type PublicUser struct {
	Id          int       `json:"id"`
	Email       string    `json:"email"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	PhoneNumber *string   `json:"phone_number"`
	Role        user.Role `json:"role"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func PublicUserFromDomain(u user.User) PublicUser {
	return PublicUser{
		Id:          u.Id,
		Email:       u.Email,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		PhoneNumber: u.PhoneNumber,
		Role:        u.UserRole,
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func PublicUsersFromDomain(us []user.User) []PublicUser {
	out := make([]PublicUser, 0, len(us))
	for _, u := range us {
		out = append(out, PublicUserFromDomain(u))
	}
	return out
}
