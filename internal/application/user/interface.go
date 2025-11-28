package userservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
)

// Service is the application-level user service contract. It lives in the
// application layer and uses application DTOs for requests/responses.
type Service interface {
	Register(ctx context.Context, req dto.RegisterRequest) (dto.PublicUser, error)
	Authenticate(ctx context.Context, req dto.LoginRequest) (dto.PublicUser, error)
	Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (dto.LoginResponse, error)

	// User management
	GetUserByID(ctx context.Context, userID int) (dto.PublicUser, error)
	UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) error
	ChangePassword(ctx context.Context, userID int, req dto.ChangePasswordRequest) error
	// ChangePasswordAdmin changes password for a user without requiring current password.
	// Intended to be used by admins.
	ChangePasswordAdmin(ctx context.Context, userID int, newPassword string) error

	// SetActiveAccount sets user's IsActive to the provided value (true = active).
	// This centralizes activate/deactivate business logic.
	SetActiveAccount(ctx context.Context, userID int, active bool) error

	// ToggleActiveAccount flips user's IsActive value (active -> inactive, inactive -> active).
	// The handler should call this to keep fetch/update logic inside the application layer.
	ToggleActiveAccount(ctx context.Context, userID int) error

	// Note: ActivateAccount/DeactivateAccount wrappers removed — use SetActiveAccount or ToggleActiveAccount.

	ListUsers(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error)
	DeleteUser(ctx context.Context, userID int) (int, error)

	// SetUserRole sets the user's role (admin-only operation). Role must be a valid domain role string.
	SetUserRole(ctx context.Context, userID int, role string) error
}
