package userservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
)

type Service interface {
	Register(ctx context.Context, req dto.RegisterRequest) (dto.PublicUser, error)
	Authenticate(ctx context.Context, req dto.LoginRequest) (dto.PublicUser, error)
	Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (dto.LoginResponse, error)

	GetUserByID(ctx context.Context, userID int) (dto.PublicUser, error)
	UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) (dto.PublicUser, error)
	ChangePassword(ctx context.Context, userID int, req dto.ChangePasswordRequest) (dto.PublicUser, error)

	ChangePasswordAdmin(ctx context.Context, userID int, newPassword string) (dto.PublicUser, error)

	ToggleActiveAccount(ctx context.Context, userID int) (dto.PublicUser, error)

	ListUsers(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error)
	DeleteUser(ctx context.Context, userID int) (int, error)

	SetUserRole(ctx context.Context, userID int, role string) (dto.PublicUser, error)
}
