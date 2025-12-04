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
	var phone *string
	if req.PhoneNumber.Defined {
		if req.PhoneNumber.Valid && req.PhoneNumber.Value != nil {
			v := strings.TrimSpace(*req.PhoneNumber.Value)
			phone = &v
		} else {

			v := ""
			phone = &v
		}
	}

	if email == "" {
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("email", email, "не может быть пустым")
	}
	if pass == "" {
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("password", nil, "не может быть пустым")
	}

	hash, err := s.hasher.Hash(pass)
	if err != nil {
		s.logger.Error("регистрация: не удалось захэшировать пароль", "err", err)
		return dto.PublicUser{}, apperrors.NewErrInternal("не удалось захэшировать пароль")
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
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &ae):
			s.logger.Info("регистрация: email уже существует", "email", email)
			return dto.PublicUser{}, apperrors.NewErrAlreadyExists("user", "email", email)
		case errors.As(err, &te):
			s.logger.Error("регистрация: превышено время ожидания репозитория", "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("регистрация: не удалось создать пользователя", "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось создать пользователя")
		}
	}

	created, err := s.repo.GetByID(ctx, id)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("регистрация: превышено время ожидания получения созданного пользователя", "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("регистрация: не удалось получить созданного пользователя", "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось получить созданного пользователя")
		}
	}
	s.logger.Info("пользователь зарегистрирован", "user_id", created.Id, "email", created.Email)

	created.PasswordHash = ""
	return dto.PublicUserFromDomain(created), nil
}

func (s *service) Authenticate(ctx context.Context, req dto.LoginRequest) (dto.PublicUser, error) {
	email := strings.TrimSpace(req.Email)
	password := strings.TrimSpace(req.Password)
	if email == "" || password == "" {
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("credentials", nil, "требуются email и пароль")
	}

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			s.logger.Info("аутентификация: пользователь не найден", "email", email)
			return dto.PublicUser{}, apperrors.NewErrNotFound("user", email)
		case errors.As(err, &te):
			s.logger.Error("аутентификация: превышено время ожидания репозитория", "email", email, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("аутентификация: не удалось получить пользователя по email", "email", email, "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("ошибка аутентификации")
		}
	}
	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		s.logger.Info("аутентификация: неверные учётные данные", "email", email)
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("password", nil, "неверные учетные данные")
	}

	if !u.IsActive {
		s.logger.Info("аутентификация: пользователь деактивирован", "email", email)
		return dto.PublicUser{}, apperrors.NewErrForbidden("аккаунт деактивирован")
	}

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
		s.logger.Error("вход: не удалось сгенерировать access токен", "user_id", u.Id, "err", err)
		return dto.LoginResponse{}, apperrors.NewErrInternal("не удалось сгенерировать токен доступа")
	}
	refresh, err := s.tokenService.GenerateRefreshToken(u.Id, refreshTTL)
	if err != nil {
		s.logger.Error("вход: не удалось сгенерировать refresh токен", "user_id", u.Id, "err", err)
		return dto.LoginResponse{}, apperrors.NewErrInternal("не удалось сгенерировать токен обновления")
	}
	s.logger.Info("пользователь вошёл", "user_id", u.Id)
	return dto.LoginResponse{User: u, AccessToken: access, RefreshToken: refresh, ExpiresAt: exp}, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	err := s.tokenService.InvalidateRefreshToken(refreshToken)
	if err != nil {
		s.logger.Error("выход: не удалось инвалидировать refresh токен", "err", err)
	} else {
		s.logger.Info("выход: токен обновления отозван")
	}
	return err
}

func (s *service) RefreshToken(ctx context.Context, refreshToken string) (dto.LoginResponse, error) {
	uid, err := s.tokenService.ValidateRefreshToken(refreshToken)
	if err != nil {
		s.logger.Info("обновление токена: недействительный токен обновления")
		return dto.LoginResponse{}, apperrors.NewErrInvalidInput("refresh_token", nil, "недействительный")
	}

	u, err := s.repo.GetByID(ctx, uid)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("обновление токена: превышено время ожидания получения пользователя", "user_id", uid, "err", err)
			return dto.LoginResponse{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("обновление токена: не удалось получить пользователя", "user_id", uid, "err", err)
			return dto.LoginResponse{}, apperrors.NewErrInternal("не удалось получить пользователя")
		}
	}

	if !u.IsActive {
		s.logger.Info("обновление токена: пользователь деактивирован", "user_id", u.Id)
		return dto.LoginResponse{}, apperrors.NewErrForbidden("аккаунт деактивирован")
	}

	accessTTL := 15 * time.Minute
	refreshTTL := 24 * time.Hour
	access, exp, err := s.tokenService.GenerateAccessToken(u.Id, string(u.UserRole), accessTTL)
	if err != nil {
		s.logger.Error("обновление токена: не удалось сгенерировать access токен", "user_id", u.Id, "err", err)
		return dto.LoginResponse{}, apperrors.NewErrInternal("не удалось сгенерировать токен доступа")
	}
	newRefresh, err := s.tokenService.GenerateRefreshToken(u.Id, refreshTTL)
	if err != nil {
		s.logger.Error("обновление токена: не удалось сгенерировать refresh токен", "user_id", u.Id, "err", err)
		return dto.LoginResponse{}, apperrors.NewErrInternal("не удалось сгенерировать токен обновления")
	}

	_ = s.tokenService.InvalidateRefreshToken(refreshToken)
	s.logger.Info("обновление токена: токены обновлены", "user_id", u.Id)
	u.PasswordHash = ""
	return dto.LoginResponse{User: dto.PublicUserFromDomain(u), AccessToken: access, RefreshToken: newRefresh, ExpiresAt: exp}, nil
}
func (s *service) UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) (dto.PublicUser, error) {

	u, err := s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return dto.PublicUser{}, apperrors.NewErrNotFound("user", req.UserID)
		case errors.As(err, &te):
			s.logger.Error("обновление профиля: превышено время ожидания получения пользователя", "user_id", req.UserID, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось получить пользователя")
		}
	}

	if req.FirstName.Defined {
		if !req.FirstName.Valid {
			u.FirstName = ""
		} else if req.FirstName.Value != nil {
			u.FirstName = strings.TrimSpace(*req.FirstName.Value)
		}
	}

	if req.LastName.Defined {
		if !req.LastName.Valid {
			u.LastName = ""
		} else if req.LastName.Value != nil {
			u.LastName = strings.TrimSpace(*req.LastName.Value)
		}
	}

	if req.PhoneNumber.Defined {
		if !req.PhoneNumber.Valid {
			v := ""
			u.PhoneNumber = &v
		} else if req.PhoneNumber.Value != nil {
			v := strings.TrimSpace(*req.PhoneNumber.Value)
			u.PhoneNumber = &v
		}
	}

	if req.Email.Defined {
		if !req.Email.Valid {
			u.Email = ""
		} else if req.Email.Value != nil {
			u.Email = strings.TrimSpace(*req.Email.Value)
		}
	}

	updated, err := s.repo.Update(ctx, u)
	if err != nil {
		var ae dberrors.ErrAlreadyExists
		var di dberrors.ErrInvalidInput
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &ae):
			return dto.PublicUser{}, apperrors.NewErrAlreadyExists("user", "email", u.Email)
		case errors.As(err, &di):
			return dto.PublicUser{}, apperrors.NewErrInvalidInput(di.Field, di.Value, di.Reason)
		case errors.As(err, &te):
			s.logger.Error("обновление профиля: превышено время ожидания репозитория", "user_id", u.Id, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("обновление профиля: не удалось обновить пользователя", "user_id", u.Id, "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось обновить пользователя")
		}
	}
	s.logger.Info("обновление профиля: обновлены поля", "user_id", updated.Id)
	updated.PasswordHash = ""
	return dto.PublicUserFromDomain(updated), nil
}

func (s *service) ChangePassword(ctx context.Context, userID int, req dto.ChangePasswordRequest) (dto.PublicUser, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return dto.PublicUser{}, apperrors.NewErrNotFound("user", userID)
		case errors.As(err, &te):
			s.logger.Error("смена пароля: превышено время ожидания получения пользователя", "user_id", userID, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось получить пользователя")
		}
	}
	if err := s.hasher.Compare(u.PasswordHash, req.CurrentPassword); err != nil {
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("current_password", nil, "текущий пароль неверный")
	}
	hash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		s.logger.Error("смена пароля: не удалось захешировать новый пароль", "user_id", u.Id, "err", err)
		return dto.PublicUser{}, apperrors.NewErrInternal("не удалось захэшировать новый пароль")
	}
	u.PasswordHash = hash
	updated, err := s.repo.Update(ctx, u)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("смена пароля: превышено время ожидания репозитория", "user_id", u.Id, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("смена пароля: не удалось обновить пользователя", "user_id", u.Id, "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось обновить пользователя")
		}
	}
	s.logger.Info("смена пароля: пароль обновлён", "user_id", u.Id)
	updated.PasswordHash = ""
	return dto.PublicUserFromDomain(updated), nil
}

func (s *service) ChangePasswordAdmin(ctx context.Context, userID int, newPassword string) (dto.PublicUser, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return dto.PublicUser{}, apperrors.NewErrNotFound("user", userID)
		case errors.As(err, &te):
			s.logger.Error("change password admin: fetch user timeout", "user_id", userID, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось получить пользователя")
		}
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		s.logger.Error("смена пароля админом: не удалось захешировать новый пароль", "user_id", u.Id, "err", err)
		return dto.PublicUser{}, apperrors.NewErrInternal("не удалось захэшировать новый пароль")
	}
	u.PasswordHash = hash
	updated, err := s.repo.Update(ctx, u)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("change password admin: repo timeout", "user_id", u.Id, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("смена пароля админом: не удалось обновить пользователя", "user_id", u.Id, "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось обновить пользователя")
		}
	}
	s.logger.Info("смена пароля админом: пароль обновлён", "user_id", u.Id)
	updated.PasswordHash = ""
	return dto.PublicUserFromDomain(updated), nil
}

func (s *service) SetUserRole(ctx context.Context, userID int, role string) (dto.PublicUser, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return dto.PublicUser{}, apperrors.NewErrNotFound("user", userID)
		case errors.As(err, &te):
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось получить пользователя")
		}
	}
	rv := strings.TrimSpace(role)
	if rv == "" {
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("role", role, "не может быть пустым")
	}
	r, err := domain.ParseRole(rv)
	if err != nil {
		return dto.PublicUser{}, apperrors.NewErrInvalidInput("role", role, "недопустимая роль")
	}
	u.UserRole = r
	updated, err := s.repo.Update(ctx, u)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("set user role: repo timeout", "user_id", u.Id, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("установка роли пользователя: не удалось обновить пользователя", "user_id", u.Id, "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось установить роль пользователя")
		}
	}
	s.logger.Info("установка роли пользователя", "user_id", u.Id, "role", u.UserRole)
	updated.PasswordHash = ""
	return dto.PublicUserFromDomain(updated), nil
}

func (s *service) ToggleActiveAccount(ctx context.Context, userID int) (dto.PublicUser, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return dto.PublicUser{}, apperrors.NewErrNotFound("user", userID)
		case errors.As(err, &te):
			s.logger.Error("toggle active: fetch user timeout", "user_id", userID, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("переключение активности: не удалось получить пользователя", "user_id", userID, "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось получить пользователя")
		}
	}

	u.IsActive = !u.IsActive
	updated, err := s.repo.Update(ctx, u)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("toggle active: repo timeout", "user_id", userID, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("переключение активности: не удалось обновить пользователя", "user_id", userID, "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось обновить пользователя")
		}
	}
	if u.IsActive {
		s.logger.Info("переключение активности: пользователь активирован", "user_id", userID)
	} else {
		s.logger.Info("переключение активности: пользователь деактивирован", "user_id", userID)
	}
	updated.PasswordHash = ""
	return dto.PublicUserFromDomain(updated), nil
}

func (s *service) ListUsers(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error) {

	dr := domain.ListRequest{Filter: req.Filter, Limit: req.Limit, Offset: req.Offset}
	users, total, err := s.repo.List(ctx, dr)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("list users: repo timeout", "err", err)
			return dto.ListUsersResponse{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("список пользователей: не удалось получить список пользователей", "err", err)
			return dto.ListUsersResponse{}, apperrors.NewErrInternal("не удалось получить список пользователей")
		}
	}

	return dto.ListUsersResponse{Users: dto.PublicUsersFromDomain(users), Total: total}, nil
}

func (s *service) GetUserByID(ctx context.Context, userID int) (dto.PublicUser, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return dto.PublicUser{}, apperrors.NewErrNotFound("user", userID)
		case errors.As(err, &te):
			s.logger.Error("get user by id: repo timeout", "user_id", userID, "err", err)
			return dto.PublicUser{}, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("получение пользователя по id: не удалось получить пользователя", "user_id", userID, "err", err)
			return dto.PublicUser{}, apperrors.NewErrInternal("не удалось получить пользователя")
		}
	}
	u.PasswordHash = ""
	return dto.PublicUserFromDomain(u), nil
}

func (s *service) DeleteUser(ctx context.Context, userID int) (int, error) {
	deletedID, err := s.repo.Delete(ctx, userID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return 0, apperrors.NewErrNotFound("user", userID)
		case errors.As(err, &te):
			s.logger.Error("delete user: repo timeout", "user_id", userID, "err", err)
			return 0, apperrors.NewErrTimeout("превышено время ожидания")
		default:
			s.logger.Error("удаление пользователя: не удалось удалить пользователя", "user_id", userID, "err", err)
			return 0, apperrors.NewErrInternal("не удалось удалить пользователя")
		}
	}
	s.logger.Info("удаление пользователя: пользователь удалён", "user_id", deletedID)
	return deletedID, nil
}
