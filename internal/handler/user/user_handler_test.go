package userhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	favdto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
	favdomain "github.com/Oleja123/estate-agency/internal/domain/favorite"
	auth "github.com/Oleja123/estate-agency/internal/handler/auth"
	"github.com/go-chi/chi/v5"
)

type mockService struct {
	ChangePasswordCalled      bool
	ChangePasswordAdminCalled bool
	LastChangePasswordUserID  int
	LastChangePasswordReq     dto.ChangePasswordRequest
	LastAdminNewPassword      string

	RegisterFunc            func(ctx context.Context, req dto.RegisterRequest) (dto.PublicUser, error)
	LoginFunc               func(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error)
	ListUsersFunc           func(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error)
	ChangePasswordFunc      func(ctx context.Context, userID int, req dto.ChangePasswordRequest) error
	GetUserByIDFunc         func(ctx context.Context, userID int) (dto.PublicUser, error)
	ToggleActiveAccountFunc func(ctx context.Context, userID int) error
	SetUserRoleFunc         func(ctx context.Context, userID int, role string) error
}

type mockFavorite struct{}

func (m *mockFavorite) Create(ctx context.Context, req favdto.CreateFavoriteRequest) (favdomain.Favorite, error) {
	return favdomain.Favorite{}, nil
}
func (m *mockFavorite) GetByUserAndProperty(ctx context.Context, key favdto.CreateFavoriteRequest) (favdomain.Favorite, error) {
	return favdomain.Favorite{}, nil
}
func (m *mockFavorite) Delete(ctx context.Context, key favdto.CreateFavoriteRequest) (int, error) {
	return 0, nil
}
func (m *mockFavorite) List(ctx context.Context, req favdto.ListFavoritesRequest) (favdto.ListFavoritesResponse, error) {
	return favdto.ListFavoritesResponse{}, nil
}
func (m *mockFavorite) Exists(ctx context.Context, key favdto.CreateFavoriteRequest) (bool, error) {
	return false, nil
}

func (m *mockService) Register(ctx context.Context, req dto.RegisterRequest) (dto.PublicUser, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, req)
	}
	return dto.PublicUser{}, nil
}
func (m *mockService) Authenticate(ctx context.Context, req dto.LoginRequest) (dto.PublicUser, error) {
	return dto.PublicUser{}, nil
}
func (m *mockService) Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, req)
	}
	return dto.LoginResponse{}, nil
}
func (m *mockService) Logout(ctx context.Context, refreshToken string) error { return nil }
func (m *mockService) RefreshToken(ctx context.Context, refreshToken string) (dto.LoginResponse, error) {
	return dto.LoginResponse{}, nil
}
func (m *mockService) GetUserByID(ctx context.Context, userID int) (dto.PublicUser, error) {
	if m.GetUserByIDFunc != nil {
		return m.GetUserByIDFunc(ctx, userID)
	}
	return dto.PublicUser{}, nil
}
func (m *mockService) UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) error {
	return nil
}
func (m *mockService) ChangePassword(ctx context.Context, userID int, req dto.ChangePasswordRequest) error {
	m.ChangePasswordCalled = true
	m.LastChangePasswordUserID = userID
	m.LastChangePasswordReq = req
	if m.ChangePasswordFunc != nil {
		return m.ChangePasswordFunc(ctx, userID, req)
	}
	return nil
}
func (m *mockService) ChangePasswordAdmin(ctx context.Context, userID int, newPassword string) error {
	m.ChangePasswordAdminCalled = true
	m.LastChangePasswordUserID = userID
	m.LastAdminNewPassword = newPassword
	return nil
}
func (m *mockService) SetActiveAccount(ctx context.Context, userID int, active bool) error {

	return nil
}

func (m *mockService) ToggleActiveAccount(ctx context.Context, userID int) error {
	if m.ToggleActiveAccountFunc != nil {
		return m.ToggleActiveAccountFunc(ctx, userID)
	}
	return nil
}
func (m *mockService) SetUserRole(ctx context.Context, userID int, role string) error {
	if m.SetUserRoleFunc != nil {
		return m.SetUserRoleFunc(ctx, userID, role)
	}
	return nil
}
func (m *mockService) ListUsers(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error) {
	if m.ListUsersFunc != nil {
		return m.ListUsersFunc(ctx, req)
	}
	return dto.ListUsersResponse{}, nil
}
func (m *mockService) DeleteUser(ctx context.Context, userID int) (int, error) { return 0, nil }

func TestHandleChangePassword_AdminPath(t *testing.T) {
	m := &mockService{}
	m.GetUserByIDFunc = func(ctx context.Context, userID int) (dto.PublicUser, error) {
		return dto.PublicUser{Id: userID}, nil
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	body := map[string]string{"new_password": "adminnew"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/users/5/password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "5")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	req = req.WithContext(auth.ContextWithUser(req.Context(), 3, "admin"))

	rr := httptest.NewRecorder()
	h.handleChangePassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got dto.PublicUser
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if got.Id != 5 {
		t.Fatalf("expected returned user id 5, got %d", got.Id)
	}
	if !m.ChangePasswordAdminCalled {
		t.Fatalf("expected ChangePasswordAdmin called")
	}
	if m.LastChangePasswordUserID != 5 {
		t.Fatalf("expected userID 5, got %d", m.LastChangePasswordUserID)
	}
}

func TestHandleChangePassword_OwnerPath(t *testing.T) {
	m := &mockService{}
	m.GetUserByIDFunc = func(ctx context.Context, userID int) (dto.PublicUser, error) {
		return dto.PublicUser{Id: userID}, nil
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	body := map[string]string{"current_password": "old", "new_password": "new"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/users/7/password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "7")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))

	req = req.WithContext(auth.ContextWithUser(req.Context(), 7, "client"))

	rr := httptest.NewRecorder()
	h.handleChangePassword(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got dto.PublicUser
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if got.Id != 7 {
		t.Fatalf("expected returned user id 7, got %d", got.Id)
	}
	if !m.ChangePasswordCalled {
		t.Fatalf("expected ChangePassword called")
	}
	if m.LastChangePasswordUserID != 7 {
		t.Fatalf("expected userID 7, got %d", m.LastChangePasswordUserID)
	}
}

func TestHandleRegister_Success(t *testing.T) {
	m := &mockService{}
	now := time.Now()
	m.RegisterFunc = func(ctx context.Context, req dto.RegisterRequest) (dto.PublicUser, error) {
		return dto.PublicUser{Id: 42, Email: req.Email, FirstName: req.FirstName, LastName: req.LastName, IsActive: true, CreatedAt: now, UpdatedAt: now}, nil
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	body := map[string]string{"email": "new@user", "password": "pwd", "first_name": "FN", "last_name": "LN"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/users/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.handleRegister(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got dto.PublicUser
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Id != 42 || got.Email != "new@user" {
		t.Fatalf("unexpected user returned: %+v", got)
	}
}

func TestHandleRegister_InvalidJSON(t *testing.T) {
	m := &mockService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	req := httptest.NewRequest(http.MethodPost, "/users/register", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.handleRegister(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLogin_Success(t *testing.T) {
	m := &mockService{}
	now := time.Now()
	m.LoginFunc = func(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
		return dto.LoginResponse{User: dto.PublicUser{Id: 7, Email: req.Email, IsActive: true, CreatedAt: now, UpdatedAt: now}, AccessToken: "at", RefreshToken: "rt", ExpiresAt: now.Add(time.Hour)}, nil
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	body := map[string]string{"email": "x@x", "password": "p"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.handleLogin(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got dto.LoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.AccessToken != "at" || got.RefreshToken != "rt" {
		t.Fatalf("unexpected tokens: %+v", got)
	}
}

func TestHandleListUsers_ReturnsList(t *testing.T) {
	m := &mockService{}
	now := time.Now()
	m.ListUsersFunc = func(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error) {
		return dto.ListUsersResponse{Users: []dto.PublicUser{{Id: 1, Email: "a@b", IsActive: true, CreatedAt: now, UpdatedAt: now}, {Id: 2, Email: "c@d", IsActive: true, CreatedAt: now, UpdatedAt: now}}, Total: 2}, nil
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rr := httptest.NewRecorder()
	h.handleListUsers(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got dto.ListUsersResponse
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 || len(got.Users) != 2 {
		t.Fatalf("unexpected list response: %+v", got)
	}
}

func TestHandleListUsers_RequiresAdmin_Middleware(t *testing.T) {
	m := &mockService{}
	now := time.Now()
	m.ListUsersFunc = func(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error) {
		return dto.ListUsersResponse{Users: []dto.PublicUser{{Id: 1, Email: "a@b", IsActive: true, CreatedAt: now, UpdatedAt: now}}, Total: 1}, nil
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Get("/", h.handleListUsers)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), 5, "client"))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), 9, "admin"))
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleChangePassword_RequiresOwnerOrAdmin_Middleware_Forbidden(t *testing.T) {
	m := &mockService{}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	r := chi.NewRouter()
	r.With(auth.RequireOwnerOrAdminMiddleware()).Patch("/{id}/password", h.handleChangePassword)

	body := map[string]string{"current_password": "old", "new_password": "new"}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPatch, "/7/password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), 6, "client"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-owner non-admin, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleRegister_AlreadyExists_Returns409(t *testing.T) {
	m := &mockService{}
	m.RegisterFunc = func(ctx context.Context, req dto.RegisterRequest) (dto.PublicUser, error) {
		return dto.PublicUser{}, apperrors.NewErrAlreadyExists("user", "email", req.Email)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	body := map[string]string{"email": "dup@x", "password": "p", "first_name": "F", "last_name": "L"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/users/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.handleRegister(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 for already exists, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleLogin_InvalidCredentials_Returns401(t *testing.T) {
	m := &mockService{}
	m.LoginFunc = func(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
		return dto.LoginResponse{}, apperrors.NewErrInvalidInput("credentials", nil, "invalid")
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	body := map[string]string{"email": "x@x", "password": "bad"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.handleLogin(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid credentials, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGetUser_NotFound_Returns404(t *testing.T) {
	m := &mockService{}

	m.ListUsersFunc = func(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error) {
		return dto.ListUsersResponse{}, apperrors.NewErrNotFound("user", 77)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rr := httptest.NewRecorder()
	h.handleListUsers(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for list users not found, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleChangePassword_Owner_BadRequest_Returns400(t *testing.T) {
	m := &mockService{}
	m.ChangePasswordFunc = func(ctx context.Context, userID int, req dto.ChangePasswordRequest) error {
		return apperrors.NewErrInvalidInput("password", nil, "bad current password")
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	body := map[string]string{"current_password": "wrong", "new_password": "new"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/users/7/password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "7")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	req = req.WithContext(auth.ContextWithUser(req.Context(), 7, "client"))
	rr := httptest.NewRecorder()

	h.handleChangePassword(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad change password, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleSetActive_TogglesToInactive(t *testing.T) {
	m := &mockService{}

	setCalled := false
	var setUserID int
	m.ToggleActiveAccountFunc = func(ctx context.Context, userID int) error {
		setCalled = true
		setUserID = userID
		return nil
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	req := httptest.NewRequest(http.MethodPatch, "/users/5/active", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "5")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	// ensure GetUserByID returns updated user for the response
	now := time.Now()
	m.GetUserByIDFunc = func(ctx context.Context, userID int) (dto.PublicUser, error) {
		return dto.PublicUser{Id: userID, Email: "u@x", IsActive: false, CreatedAt: now, UpdatedAt: now}, nil
	}

	h.handleSetActive(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got dto.PublicUser
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if got.Id != 5 {
		t.Fatalf("expected returned user id 5, got %d", got.Id)
	}
	if !setCalled || setUserID != 5 {
		t.Fatalf("expected ToggleActiveAccount called with user 5, got called=%v user=%d", setCalled, setUserID)
	}
}

func TestHandleSetActive_TogglesToActive(t *testing.T) {
	m := &mockService{}
	setCalled := false
	var setUserID int
	m.ToggleActiveAccountFunc = func(ctx context.Context, userID int) error {
		setCalled = true
		setUserID = userID
		return nil
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	h := NewUserHandler(m, logger, &mockFavorite{})

	req := httptest.NewRequest(http.MethodPatch, "/users/8/active", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "8")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	now := time.Now()
	m.GetUserByIDFunc = func(ctx context.Context, userID int) (dto.PublicUser, error) {
		return dto.PublicUser{Id: userID, Email: "u@x", IsActive: true, CreatedAt: now, UpdatedAt: now}, nil
	}

	h.handleSetActive(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got dto.PublicUser
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	if got.Id != 8 {
		t.Fatalf("expected returned user id 8, got %d", got.Id)
	}
	if !setCalled || setUserID != 8 {
		t.Fatalf("expected ToggleActiveAccount called with user 8, got called=%v user=%d", setCalled, setUserID)
	}
}
