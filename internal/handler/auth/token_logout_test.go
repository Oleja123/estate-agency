package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"log/slog"

	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
)

type mockUserSvc struct{
	LogoutFunc func(ctx context.Context, refreshToken string) error
}

func (m *mockUserSvc) Register(ctx context.Context, req dto.RegisterRequest) (dto.PublicUser, error) { return dto.PublicUser{}, nil }
func (m *mockUserSvc) Authenticate(ctx context.Context, req dto.LoginRequest) (dto.PublicUser, error) { return dto.PublicUser{}, nil }
func (m *mockUserSvc) Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) { return dto.LoginResponse{}, nil }
func (m *mockUserSvc) Logout(ctx context.Context, refreshToken string) error {
	if m.LogoutFunc != nil { return m.LogoutFunc(ctx, refreshToken) }
	return nil
}
func (m *mockUserSvc) RefreshToken(ctx context.Context, refreshToken string) (dto.LoginResponse, error) { return dto.LoginResponse{}, nil }
func (m *mockUserSvc) GetUserByID(ctx context.Context, userID int) (dto.PublicUser, error) { return dto.PublicUser{}, nil }
func (m *mockUserSvc) UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) (dto.PublicUser, error) { return dto.PublicUser{}, nil }
func (m *mockUserSvc) ChangePassword(ctx context.Context, userID int, req dto.ChangePasswordRequest) (dto.PublicUser, error) { return dto.PublicUser{}, nil }
func (m *mockUserSvc) ChangePasswordAdmin(ctx context.Context, userID int, newPassword string) (dto.PublicUser, error) { return dto.PublicUser{}, nil }
func (m *mockUserSvc) ToggleActiveAccount(ctx context.Context, userID int) (dto.PublicUser, error) { return dto.PublicUser{}, nil }
func (m *mockUserSvc) ListUsers(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error) { return dto.ListUsersResponse{}, nil }
func (m *mockUserSvc) DeleteUser(ctx context.Context, userID int) (int, error) { return 0, nil }
func (m *mockUserSvc) SetUserRole(ctx context.Context, userID int, role string) (dto.PublicUser, error) { return dto.PublicUser{}, nil }

func logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestHandleLogout_Success(t *testing.T) {
	m := &mockUserSvc{}
	var gotToken string
	m.LogoutFunc = func(ctx context.Context, refreshToken string) error {
		gotToken = refreshToken
		return nil
	}
	h := NewTokenHandler(nil, m)

	b, _ := json.Marshal(map[string]string{"refresh_token": "rtoken"})
	req := httptest.NewRequest(http.MethodPost, "/tokens/logout", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	rc := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	rr := httptest.NewRecorder()
	h.handleLogout(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body map[string]string
	err := json.NewDecoder(rr.Body).Decode(&body)
	require.NoError(t, err)
	require.Equal(t, "logged out", body["message"])
	require.Equal(t, "rtoken", gotToken)
}

func TestHandleLogout_BadJSON(t *testing.T) {
	m := &mockUserSvc{}
	h := NewTokenHandler(nil, m)

	req := httptest.NewRequest(http.MethodPost, "/tokens/logout", bytes.NewReader([]byte("not-json")))
	req.Header.Set("Content-Type", "application/json")

	rc := chi.NewRouteContext()
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	rr := httptest.NewRecorder()
	h.handleLogout(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}
