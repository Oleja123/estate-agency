package userservice

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	token "github.com/Oleja123/estate-agency/internal/application/token"
	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
	pwd "github.com/Oleja123/estate-agency/internal/application/user/password"
	optional "github.com/denpa16/optional-go-type"

	domain "github.com/Oleja123/estate-agency/internal/domain/user"
	basedberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	CreateFunc   func(ctx context.Context, u domain.User) (int, error)
	GetByEmailFn func(ctx context.Context, email string) (domain.User, error)
	UpdateFunc   func(ctx context.Context, u domain.User) error
	DeleteFunc   func(ctx context.Context, id int) (int, error)
	GetByIDFunc  func(ctx context.Context, id int) (domain.User, error)
	ListFunc     func(ctx context.Context, req domain.ListRequest) ([]domain.User, int, error)
}

func (m *mockRepo) Create(ctx context.Context, u domain.User) (int, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, u)
	}
	return 0, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id int) (domain.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return domain.User{}, basedberrors.NewErrNotFound("user", id)
}

func (m *mockRepo) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, email)
	}
	return domain.User{}, basedberrors.NewErrNotFound("user", email)
}

func (m *mockRepo) Update(ctx context.Context, u domain.User) (domain.User, error) {
	if m.UpdateFunc != nil {
		return u, m.UpdateFunc(ctx, u)
	}
	return u, nil
}

func (m *mockRepo) Delete(ctx context.Context, id int) (int, error) {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return id, nil
}

func (m *mockRepo) List(ctx context.Context, req domain.ListRequest) ([]domain.User, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return nil, 0, nil
}

func makeLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestUserService_Register(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()
	tests := []struct {
		name        string
		repoFactory func() *mockRepo
		req         dto.RegisterRequest
		wantErr     bool
		wantErrType string
		wantID      int
	}{
		{
			name: "success",
			repoFactory: func() *mockRepo {
				return &mockRepo{
					GetByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
						return domain.User{}, basedberrors.NewErrNotFound("user", email)
					},
					CreateFunc: func(ctx context.Context, u domain.User) (int, error) {

						if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("secret")); err != nil {
							return 0, err
						}
						if u.PhoneNumber == nil || *u.PhoneNumber != "100500" {
							var got string
							if u.PhoneNumber != nil {
								got = *u.PhoneNumber
							}
							return 0, fmt.Errorf("phone mismatch: %s", got)
						}
						return 11, nil
					},
					GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
						return domain.User{Id: id, Email: "a@b.com"}, nil
					},
				}
			},
			req:     dto.RegisterRequest{Email: "a@b.com", Password: "secret", PhoneNumber: optional.OptionalString{Defined: true, Valid: true, Value: func() *string { v := "100500"; return &v }()}},
			wantErr: false,
			wantID:  11,
		},
		{
			name:        "empty_email",
			repoFactory: func() *mockRepo { return &mockRepo{} },
			req:         dto.RegisterRequest{Email: "", Password: "p"},
			wantErr:     true, wantErrType: "invalid",
		},
		{
			name:        "empty_password",
			repoFactory: func() *mockRepo { return &mockRepo{} },
			req:         dto.RegisterRequest{Email: "x@y.com", Password: ""},
			wantErr:     true, wantErrType: "invalid",
		},
		{
			name: "already_exists",
			repoFactory: func() *mockRepo {
				return &mockRepo{
					CreateFunc: func(ctx context.Context, u domain.User) (int, error) {
						return 0, basedberrors.NewErrAlreadyExists("user", "email", u.Email)
					},
				}
			},
			req:     dto.RegisterRequest{Email: "dup@e.com", Password: "p"},
			wantErr: true, wantErrType: "already",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := tc.repoFactory()
			svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

			user, err := svc.Register(ctx, tc.req)
			if tc.wantErr {
				require.Error(t, err)
				switch tc.wantErrType {
				case "invalid":
					_, ok := err.(apperrors.ErrInvalidInput)
					assert.True(t, ok)
				case "already":
					_, ok := err.(apperrors.ErrAlreadyExists)
					assert.True(t, ok)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantID, user.Id)
			}
		})
	}
}

func TestUserService_Login(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	tests := []struct {
		name        string
		repoFactory func() *mockRepo
		req         dto.LoginRequest
		wantErr     bool
	}{
		{
			name: "success",
			repoFactory: func() *mockRepo {

				h, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
				return &mockRepo{
					GetByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
						return domain.User{Id: 1, Email: email, PasswordHash: string(h), IsActive: true}, nil
					},
				}
			},
			req:     dto.LoginRequest{Email: "a@b.com", Password: "secret"},
			wantErr: false,
		},
		{
			name: "wrong_password",
			repoFactory: func() *mockRepo {
				return &mockRepo{GetByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
					return domain.User{Id: 1, Email: email, PasswordHash: "x"}, nil
				}}
			},
			req:     dto.LoginRequest{Email: "a@b.com", Password: "secret"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := tc.repoFactory()
			svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

			_, err := svc.Authenticate(ctx, tc.req)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUserService_ChangePassword(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	h, err := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, PasswordHash: string(h)}, nil
		},
		UpdateFunc: func(ctx context.Context, u domain.User) error {

			if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("newpass")); err != nil {
				return err
			}
			return nil
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())
	_, err = svc.ChangePassword(ctx, 1, dto.ChangePasswordRequest{CurrentPassword: "oldpass", NewPassword: "newpass"})
	require.NoError(t, err)
}

func TestUserService_LoginWithTokens(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	h, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &mockRepo{
		GetByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
			return domain.User{Id: 42, Email: email, PasswordHash: string(h), IsActive: true}, nil
		},
	}
	tokSvc := token.NewMemoryService()
	svc := New(repo, logger, pwd.NewBcryptHasher(), tokSvc)

	resp, err := svc.Login(ctx, dto.LoginRequest{Email: "a@b.com", Password: "secret"})
	require.NoError(t, err)
	require.NotEmpty(t, resp.AccessToken)
	require.NotEmpty(t, resp.RefreshToken)

	uid, _, err := tokSvc.ValidateAccessToken(resp.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, 42, uid)
}

func TestUserService_RefreshAndLogout(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	h, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &mockRepo{
		GetByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
			return domain.User{Id: 7, Email: email, PasswordHash: string(h), IsActive: true}, nil
		},
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, Email: "x@x.com", IsActive: true}, nil
		},
	}
	tokSvc := token.NewMemoryService()
	svc := New(repo, logger, pwd.NewBcryptHasher(), tokSvc)

	resp, err := svc.Login(ctx, dto.LoginRequest{Email: "x@x.com", Password: "secret"})
	require.NoError(t, err)

	newResp, err := svc.RefreshToken(ctx, resp.RefreshToken)
	require.NoError(t, err)
	require.NotEmpty(t, newResp.AccessToken)
	require.NotEmpty(t, newResp.RefreshToken)

	_, err = tokSvc.ValidateRefreshToken(resp.RefreshToken)
	assert.Error(t, err)

	err = svc.Logout(ctx, newResp.RefreshToken)
	require.NoError(t, err)
	_, err = tokSvc.ValidateRefreshToken(newResp.RefreshToken)
	assert.Error(t, err)
}

func TestUserService_RefreshToken_Deactivated(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	h, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &mockRepo{
		GetByEmailFn: func(ctx context.Context, email string) (domain.User, error) {
			return domain.User{Id: 7, Email: email, PasswordHash: string(h), IsActive: true}, nil
		},
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, Email: "x@x.com", IsActive: false}, nil
		},
	}
	tokSvc := token.NewMemoryService()
	svc := New(repo, logger, pwd.NewBcryptHasher(), tokSvc)

	resp, err := svc.Login(ctx, dto.LoginRequest{Email: "x@x.com", Password: "secret"})
	require.NoError(t, err)

	_, err = svc.RefreshToken(ctx, resp.RefreshToken)
	require.Error(t, err)
	_, ok := err.(apperrors.ErrForbidden)
	assert.True(t, ok)
}

func TestUserService_UpdateProfile_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, Email: "a@b.com"}, nil
		},
		UpdateFunc: func(ctx context.Context, u domain.User) error {
			return basedberrors.NewErrAlreadyExists("user", "email", u.Email)
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

	req := dto.UpdateProfileRequest{
		UserID: 1,
		Email:  optional.OptionalString{Defined: true, Valid: true, Value: func() *string { v := "dup@e.com"; return &v }()},
	}

	_, err := svc.UpdateProfile(ctx, req)
	require.Error(t, err)
	_, ok := err.(apperrors.ErrAlreadyExists)
	assert.True(t, ok)
}

func TestUserService_ChangePassword_WrongCurrent(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	h, err := bcrypt.GenerateFromPassword([]byte("oldpass"), bcrypt.DefaultCost)
	require.NoError(t, err)

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, PasswordHash: string(h)}, nil
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

	_, err = svc.ChangePassword(ctx, 1, dto.ChangePasswordRequest{CurrentPassword: "wrong", NewPassword: "newpass"})
	require.Error(t, err)
	_, ok := err.(apperrors.ErrInvalidInput)
	assert.True(t, ok)
}

func TestUserService_DeactivateAccount_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{}, basedberrors.NewErrNotFound("user", id)
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

	_, err := svc.ToggleActiveAccount(ctx, 99)
	require.Error(t, err)
	_, ok := err.(apperrors.ErrNotFound)
	assert.True(t, ok)
}

func TestUserService_DeleteUser_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		DeleteFunc: func(ctx context.Context, id int) (int, error) {
			return id, basedberrors.NewErrNotFound("user", id)
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

	_, err := svc.DeleteUser(ctx, 77)
	require.Error(t, err)
	_, ok := err.(apperrors.ErrNotFound)
	assert.True(t, ok)
}

func TestUserService_ListUsers_Pagination(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		ListFunc: func(ctx context.Context, req domain.ListRequest) ([]domain.User, int, error) {
			users := []domain.User{
				{Id: 1, Email: "a@b.com"},
				{Id: 2, Email: "b@b.com"},
			}
			return users, 5, nil
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

	resp, err := svc.ListUsers(ctx, dto.ListUsersRequest{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, 5, resp.Total)
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{}, basedberrors.NewErrNotFound("user", id)
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

	_, err := svc.GetUserByID(ctx, 42)
	require.Error(t, err)
	_, ok := err.(apperrors.ErrNotFound)
	assert.True(t, ok)
}

func TestUserService_UpdateProfile_Success(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, Email: "a@b.com", FirstName: "", LastName: "", PhoneNumber: nil, UserRole: domain.RoleClient}, nil
		},
		UpdateFunc: func(ctx context.Context, u domain.User) error {

			if u.FirstName != "John" {
				return fmt.Errorf("first name not updated: %s", u.FirstName)
			}
			if u.LastName != "Doe" {
				return fmt.Errorf("last name not updated: %s", u.LastName)
			}
			if u.PhoneNumber == nil || *u.PhoneNumber != "123" {
				var got string
				if u.PhoneNumber != nil {
					got = *u.PhoneNumber
				}
				return fmt.Errorf("phone not updated: %s", got)
			}
			if u.Email != "new@e.com" {
				return fmt.Errorf("email not updated: %s", u.Email)
			}

			return nil
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

	req := dto.UpdateProfileRequest{
		UserID:      1,
		FirstName:   optional.OptionalString{Defined: true, Valid: true, Value: func() *string { v := "John"; return &v }()},
		LastName:    optional.OptionalString{Defined: true, Valid: true, Value: func() *string { v := "Doe"; return &v }()},
		PhoneNumber: optional.OptionalString{Defined: true, Valid: true, Value: func() *string { v := "123"; return &v }()},
		Email:       optional.OptionalString{Defined: true, Valid: true, Value: func() *string { v := "new@e.com"; return &v }()},
		Role:        optional.OptionalString{Defined: true, Valid: true, Value: func() *string { v := "admin"; return &v }()},
	}

	_, err := svc.UpdateProfile(ctx, req)
	require.NoError(t, err)
}

func TestUserService_UpdateProfile_Partial(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			p := "P"
			return domain.User{Id: id, Email: "a@b.com", FirstName: "", LastName: "L", PhoneNumber: &p, UserRole: domain.RoleClient}, nil
		},
		UpdateFunc: func(ctx context.Context, u domain.User) error {
			if u.FirstName != "OnlyFirst" {
				return fmt.Errorf("first name not updated: %s", u.FirstName)
			}

			if u.LastName != "L" {
				return fmt.Errorf("last name was modified: %s", u.LastName)
			}
			if u.Email != "a@b.com" {
				return fmt.Errorf("email was modified: %s", u.Email)
			}
			return nil
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

	req := dto.UpdateProfileRequest{
		UserID:    1,
		FirstName: optional.OptionalString{Defined: true, Valid: true, Value: func() *string { v := "OnlyFirst"; return &v }()},
	}

	_, err := svc.UpdateProfile(ctx, req)
	require.NoError(t, err)
}

func TestUserService_SetUserRole_Success(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, Email: "a@b.com", UserRole: domain.RoleClient}, nil
		},
		UpdateFunc: func(ctx context.Context, u domain.User) error {
			if u.UserRole != domain.RoleAdmin {
				return fmt.Errorf("role not updated: %s", u.UserRole)
			}
			return nil
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())

	_, err := svc.SetUserRole(ctx, 1, "admin")
	require.NoError(t, err)
}

func TestUserService_DeactivateAccount_Success(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, IsActive: true}, nil
		},
		UpdateFunc: func(ctx context.Context, u domain.User) error {
			if u.IsActive {
				return fmt.Errorf("user still active")
			}
			return nil
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())
	_, err := svc.ToggleActiveAccount(ctx, 5)
	require.NoError(t, err)
}

func TestUserService_ChangePasswordAdmin_RepoAlreadyExists(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id}, nil
		},
		UpdateFunc: func(ctx context.Context, u domain.User) error {
			return basedberrors.NewErrAlreadyExists("user", "email", "dup@e.com")
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())
	_, err := svc.ChangePasswordAdmin(ctx, 1, "newpass")
	require.Error(t, err)
	_, ok := err.(apperrors.ErrInternal)
	assert.True(t, ok)
}

func TestUserService_SetUserRole_RepoAlreadyExists(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, UserRole: domain.RoleClient}, nil
		},
		UpdateFunc: func(ctx context.Context, u domain.User) error {
			return basedberrors.NewErrAlreadyExists("user", "email", "dup@e.com")
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())
	_, err := svc.SetUserRole(ctx, 1, "admin")
	require.Error(t, err)
	_, ok := err.(apperrors.ErrInternal)
	assert.True(t, ok)
}

func TestUserService_GetUserByID_Success(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		GetByIDFunc: func(ctx context.Context, id int) (domain.User, error) {
			return domain.User{Id: id, Email: "found@e.com"}, nil
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())
	u, err := svc.GetUserByID(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 10, u.Id)
	assert.Equal(t, "found@e.com", u.Email)
}

func TestUserService_DeleteUser_Success(t *testing.T) {
	ctx := context.Background()
	logger := makeLogger()

	repo := &mockRepo{
		DeleteFunc: func(ctx context.Context, id int) (int, error) {
			return id, nil
		},
	}

	svc := New(repo, logger, pwd.NewBcryptHasher(), token.NewMemoryService())
	did, err := svc.DeleteUser(ctx, 7)
	require.NoError(t, err)
	assert.Equal(t, 7, did)
}
