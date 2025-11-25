package user

import "context"

type Service interface {
	Register(ctx context.Context, req RegisterRequest) (User, error)
	Login(ctx context.Context, req LoginRequest) (LoginResponse, error)
	Logout(ctx context.Context, userID int) error
	RefreshToken(ctx context.Context, refreshToken string) (LoginResponse, error)

	GetProfile(ctx context.Context, userID int) (User, error)
	UpdateProfile(ctx context.Context, req UpdateProfileRequest) error
	ChangePassword(ctx context.Context, req ChangePasswordRequest) error
	DeactivateAccount(ctx context.Context, userID int) error

	ListUsers(ctx context.Context, req ListUsersRequest) (ListUsersResponse, error)
	GetUserByID(ctx context.Context, userID, requesterID int) (User, error)
	UpdateUserRole(ctx context.Context, req UpdateUserRoleRequest) error
	DeleteUser(ctx context.Context, userID, requesterID int) error

	RequestPasswordReset(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
}
