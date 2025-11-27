package userservice

import (
	"context"
	"errors"
	"strings"
	"time"

	"log/slog"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	token "github.com/Oleja123/estate-agency/internal/application/token"
	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
	password "github.com/Oleja123/estate-agency/internal/application/user/password"
	domain "github.com/Oleja123/estate-agency/internal/domain/user"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
)

// Ensure service implements application Service at compile time.
var _ Service = (*service)(nil)

type service struct {
	repo         domain.Repository
	logger       *slog.Logger
	hasher       password.Hasher
	tokenService token.Service
}

func New(repo domain.Repository, logger *slog.Logger, hasher password.Hasher, tokenService token.Service) Service {
	return &service{repo: repo, logger: logger, hasher: hasher, tokenService: tokenService}
}

func (s *service) Register(ctx context.Context, req dto.RegisterRequest) (dto.PublicUser, error) {
	email := strings.TrimSpace(req.Email)
	pass := strings.TrimSpace(req.Password)
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	phone := ""
	if req.PhoneNumber.Defined {
		if req.PhoneNumber.Valid && req.PhoneNumber.Value != nil {
			phone = strings.TrimSpace(*req.PhoneNumber.Value)
		}
	}

	if email == "" {
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("email", email, "must not be empty")
	}
	if pass == "" {
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("password", nil, "must not be empty")
	}

	hash, err := s.hasher.Hash(pass)
	if err != nil {
		s.logger.Error("register: failed to hash password", "err", err)
		return dto.PublicUser{}, apperrors.NewErrInternal("failed to hash password")
	}

	u := domain.User{
		Email:        email,
		PasswordHash: hash,
		FirstName:    firstName,
		LastName:     lastName,
		PhoneNumber:  phone,
		UserRole:     domain.RoleClient,
	}

	id, err := s.repo.Create(ctx, u)
	if err != nil {
		var ae dberrors.ErrAlreadyExists
		if errors.As(err, &ae) {
			s.logger.Info("register: email already exists", "email", email)
			return dto.PublicUser{}, apperrors.NewErrAlreadyExists("user", "email", email)
		}
		s.logger.Error("register: failed to create user", "err", err)
		return dto.PublicUser{}, apperrors.NewErrInternal("failed to create user")
	}

	created, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("register: failed to fetch created user", "err", err)
		return dto.PublicUser{}, apperrors.NewErrInternal("failed to fetch created user")
	}
	s.logger.Info("user registered", "user_id", created.Id, "email", created.Email)
	// Do not leak password hash outside the service layer and map to PublicUser
	created.PasswordHash = ""
	return dto.PublicUserFromDomain(created), nil
}

func (s *service) Authenticate(ctx context.Context, req dto.LoginRequest) (dto.PublicUser, error) {
	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)
	if email == "" || password == "" {
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("credentials", nil, "email and password required")
	}

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			s.logger.Info("authenticate: user not found", "email", email)
			return dto.PublicUser{}, apperrors.NewErrNotFound("user", email)
		}
		s.logger.Error("authenticate: failed to get user by email", "email", email, "err", err)
		return dto.PublicUser{}, apperrors.NewErrInternal("authenticate failed")
	}
	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		s.logger.Info("authenticate: invalid credentials", "email", email)
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("password", nil, "invalid credentials")
	}
	// Do not return password hash and map to PublicUser
	u.PasswordHash = ""
	return dto.PublicUserFromDomain(u), nil
}

func (s *service) Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	u, err := s.Authenticate(ctx, req)
	if err != nil {
		return dto.LoginResponse{}, err
	}

	accessTTL := 15 * time.Minute
	refreshTTL := 24 * time.Hour
	access, exp, err := s.tokenService.GenerateAccessToken(u.Id, string(u.Role), accessTTL)
	if err != nil {
		s.logger.Error("login: failed to generate access token", "user_id", u.Id, "err", err)
		return dto.LoginResponse{}, apperrors.NewErrInternal("failed to generate access token")
	}
	refresh, err := s.tokenService.GenerateRefreshToken(u.Id, refreshTTL)
	if err != nil {
		s.logger.Error("login: failed to generate refresh token", "user_id", u.Id, "err", err)
		return dto.LoginResponse{}, apperrors.NewErrInternal("failed to generate refresh token")
	}
	s.logger.Info("user logged in", "user_id", u.Id)
	return dto.LoginResponse{User: u, AccessToken: access, RefreshToken: refresh, ExpiresAt: exp}, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	err := s.tokenService.InvalidateRefreshToken(refreshToken)
	if err != nil {
		s.logger.Error("logout: failed to invalidate refresh token", "err", err)
	} else {
		s.logger.Info("logout: refresh token invalidated")
	}
	return err
}

func (s *service) RefreshToken(ctx context.Context, refreshToken string) (dto.LoginResponse, error) {
	uid, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		s.logger.Info("refresh: invalid refresh token")
		return dto.LoginResponse{}, apperrors.NewErrInvalidInput("refresh_token", nil, "invalid")
	}
	// fetch user
	u, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		s.logger.Error("refresh: failed to fetch user", "user_id", uid, "err", err)
		return dto.LoginResponse{}, apperrors.NewErrInternal("failed to fetch user")
	}
	// rotate tokens
	accessTTL := 15 * time.Minute
	refreshTTL := 24 * time.Hour
	access, exp, err := s.tokenService.GenerateAccessToken(u.Id, string(u.UserRole), accessTTL)
	if err != nil {
		s.logger.Error("refresh: failed to generate access token", "user_id", u.Id, "err", err)
		return dto.LoginResponse{}, apperrors.NewErrInternal("failed to generate access token")
	}
	newRefresh, err := s.tokenService.GenerateRefreshToken(u.Id, refreshTTL)
	if err != nil {
		s.logger.Error("refresh: failed to generate refresh token", "user_id", u.Id, "err", err)
		return dto.LoginResponse{}, apperrors.NewErrInternal("failed to generate refresh token")
	}
	// invalidate old refresh
	_ = s.tokenService.InvalidateRefreshToken(refreshToken)
	s.logger.Info("refresh: tokens rotated", "user_id", u.Id)
	u.PasswordHash = ""
	return dto.LoginResponse{User: dto.PublicUserFromDomain(u), AccessToken: access, RefreshToken: newRefresh, ExpiresAt: exp}, nil
}
func (s *service) UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) error {
	// fetch existing
	u, err := s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return apperrors.NewErrNotFound("user", req.UserID)
		}
		return apperrors.NewErrInternal("failed to fetch user")
	}

	// FirstName
	if req.FirstName.Defined {
		if !req.FirstName.Valid {
			u.FirstName = ""
		} else if req.FirstName.Value != nil {
			u.FirstName = strings.TrimSpace(*req.FirstName.Value)
		}
	}

	// LastName
	if req.LastName.Defined {
		if !req.LastName.Valid {
			u.LastName = ""
		} else if req.LastName.Value != nil {
			u.LastName = strings.TrimSpace(*req.LastName.Value)
		}
	}

	// PhoneNumber
	if req.PhoneNumber.Defined {
		if !req.PhoneNumber.Valid {
			u.PhoneNumber = ""
		} else if req.PhoneNumber.Value != nil {
			u.PhoneNumber = strings.TrimSpace(*req.PhoneNumber.Value)
		}
	}

	// Email - assign if provided; uniqueness check is delegated to repository
	if req.Email.Defined {
		if !req.Email.Valid {
			u.Email = ""
		} else if req.Email.Value != nil {
			u.Email = strings.TrimSpace(*req.Email.Value)
		}
	}

	// Role (optional string -> domain.Role)
	if req.Role.Defined {
		if !req.Role.Valid {
			u.UserRole = domain.Role("")
		} else if req.Role.Value != nil {
			rv := strings.TrimSpace(*req.Role.Value)
			if rv == "" {
				u.UserRole = domain.Role("")
			} else {
				// parse/validate role via domain helper
				r, err := domain.ParseRole(rv)
				if err != nil {
					return apperrors.NewErrInvalidInput("role", rv, "invalid role")
				}
				u.UserRole = r
			}
		}
	}

	err = s.repo.Update(ctx, u)
	if err != nil {
		var ae dberrors.ErrAlreadyExists
		if errors.As(err, &ae) {
			return apperrors.NewErrAlreadyExists("user", "email", u.Email)
		}
		s.logger.Error("update profile: failed to update user", "user_id", u.Id, "err", err)
		return apperrors.NewErrInternal("failed to update user")
	}
	s.logger.Info("update profile: updated fields", "user_id", u.Id)
	return nil
}

func (s *service) ChangePassword(ctx context.Context, userID int, req dto.ChangePasswordRequest) error {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return apperrors.NewErrNotFound("user", userID)
		}
		return apperrors.NewErrInternal("failed to fetch user")
	}
	if err := s.hasher.Compare(u.PasswordHash, req.CurrentPassword); err != nil {
		return apperrors.NewErrInvalidInput("current_password", nil, "invalid")
	}
	hash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		s.logger.Error("change password: failed to hash new password", "user_id", u.Id, "err", err)
		return apperrors.NewErrInternal("failed to hash new password")
	}
	u.PasswordHash = hash
	err = s.repo.Update(ctx, u)
	if err != nil {
		var ae dberrors.ErrAlreadyExists
		if errors.As(err, &ae) {
			// unlikely for password change but map anyway
			return apperrors.NewErrAlreadyExists("user", "password", "")
		}
		s.logger.Error("change password: failed to update user", "user_id", u.Id, "err", err)
		return apperrors.NewErrInternal("failed to update user")
	}
	s.logger.Info("change password: password updated", "user_id", u.Id)
	return nil
}

// ChangePasswordAdmin allows an admin to change a user's password without the
// current password. This performs hashing and updates the repository.
func (s *service) ChangePasswordAdmin(ctx context.Context, userID int, newPassword string) error {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return apperrors.NewErrNotFound("user", userID)
		}
		return apperrors.NewErrInternal("failed to fetch user")
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		s.logger.Error("change password admin: failed to hash new password", "user_id", u.Id, "err", err)
		return apperrors.NewErrInternal("failed to hash new password")
	}
	u.PasswordHash = hash
	if err := s.repo.Update(ctx, u); err != nil {
		s.logger.Error("change password admin: failed to update user", "user_id", u.Id, "err", err)
		return apperrors.NewErrInternal("failed to update user")
	}
	s.logger.Info("change password admin: password updated", "user_id", u.Id)
	return nil
}

func (s *service) DeactivateAccount(ctx context.Context, userID int) error {
	return s.SetActiveAccount(ctx, userID, false)
}

// ActivateAccount sets IsActive=true for the specified user.
func (s *service) ActivateAccount(ctx context.Context, userID int) error {
	return s.SetActiveAccount(ctx, userID, true)
}

// SetActiveAccount sets the user's IsActive flag to the given value and
// persists the change. Centralized implementation for activation/deactivation.
func (s *service) SetActiveAccount(ctx context.Context, userID int, active bool) error {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return apperrors.NewErrNotFound("user", userID)
		}
		s.logger.Error("set active: failed to fetch user", "user_id", userID, "err", err)
		return apperrors.NewErrInternal("failed to fetch user")
	}
	u.IsActive = active
	err = s.repo.Update(ctx, u)
	if err != nil {
		s.logger.Error("set active: failed to update user", "user_id", userID, "err", err)
		return apperrors.NewErrInternal("failed to update user")
	}
	if active {
		s.logger.Info("set active: user activated", "user_id", userID)
	} else {
		s.logger.Info("set active: user deactivated", "user_id", userID)
	}
	return nil
}

// ToggleActiveAccount flips the user's IsActive flag and persists the change.
// This keeps the fetching and update logic inside the application layer so handlers stay thin.
func (s *service) ToggleActiveAccount(ctx context.Context, userID int) error {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return apperrors.NewErrNotFound("user", userID)
		}
		s.logger.Error("toggle active: failed to fetch user", "user_id", userID, "err", err)
		return apperrors.NewErrInternal("failed to fetch user")
	}

	u.IsActive = !u.IsActive
	if err := s.repo.Update(ctx, u); err != nil {
		s.logger.Error("toggle active: failed to update user", "user_id", userID, "err", err)
		return apperrors.NewErrInternal("failed to update user")
	}
	if u.IsActive {
		s.logger.Info("toggle active: user activated", "user_id", userID)
	} else {
		s.logger.Info("toggle active: user deactivated", "user_id", userID)
	}
	return nil
}

func (s *service) ListUsers(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error) {
	// convert dto.ListUsersRequest -> domain.ListRequest
	dr := domain.ListRequest{Filter: req.Filter, Limit: req.Limit, Offset: req.Offset}
	users, total, err := s.repo.List(ctx, dr)
	if err != nil {
		s.logger.Error("list users: failed to list users", "err", err)
		return dto.ListUsersResponse{}, apperrors.NewErrInternal("failed to list users")
	}
	// sanitize users - do not expose password hashes and map to public DTO
	for i := range users {
		users[i].PasswordHash = ""
	}
	return dto.ListUsersResponse{Users: dto.PublicUsersFromDomain(users), Total: total}, nil
}

func (s *service) GetUserByID(ctx context.Context, userID int) (dto.PublicUser, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return dto.PublicUser{}, apperrors.NewErrNotFound("user", userID)
		}
		s.logger.Error("get user by id: failed to fetch user", "user_id", userID, "err", err)
		return dto.PublicUser{}, apperrors.NewErrInternal("failed to fetch user")
	}
	u.PasswordHash = ""
	return dto.PublicUserFromDomain(u), nil
}

func (s *service) DeleteUser(ctx context.Context, userID int) (int, error) {
	deletedID, err := s.repo.Delete(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return 0, apperrors.NewErrNotFound("user", userID)
		}
		s.logger.Error("delete user: failed to delete user", "user_id", userID, "err", err)
		return 0, apperrors.NewErrInternal("failed to delete user")
	}
	s.logger.Info("delete user: user deleted", "user_id", deletedID)
	return deletedID, nil
}
