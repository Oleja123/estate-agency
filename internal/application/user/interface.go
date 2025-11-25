package userservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
	"github.com/Oleja123/estate-agency/internal/domain/user"
)

// Service is the application-level user service contract. It lives in the
// application layer and uses application DTOs for requests/responses.
type Service interface {
	Register(ctx context.Context, req dto.RegisterRequest) (user.User, error)
	Authenticate(ctx context.Context, req dto.LoginRequest) (user.User, error)
	Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (dto.LoginResponse, error)

	// User management
	GetUserByID(ctx context.Context, userID, requesterID int) (user.User, error)
	UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) error
	ChangePassword(ctx context.Context, req dto.ChangePasswordRequest) error
	DeactivateAccount(ctx context.Context, userID int) error

	ListUsers(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error)
	DeleteUser(ctx context.Context, userID, requesterID int) error
}
