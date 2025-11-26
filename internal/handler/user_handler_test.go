package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
	"github.com/go-chi/chi/v5"
)

// mockService implements usersvc.Service with minimal stubs for testing handler behavior.
type mockService struct {
	ChangePasswordCalled      bool
	ChangePasswordAdminCalled bool
	LastChangePasswordUserID  int
	LastChangePasswordReq     dto.ChangePasswordRequest
	LastAdminNewPassword      string
}

func (m *mockService) Register(ctx context.Context, req dto.RegisterRequest) (dto.PublicUser, error) {
	return dto.PublicUser{}, nil
}
func (m *mockService) Authenticate(ctx context.Context, req dto.LoginRequest) (dto.PublicUser, error) {
	return dto.PublicUser{}, nil
}
func (m *mockService) Login(ctx context.Context, req dto.LoginRequest) (dto.LoginResponse, error) {
	return dto.LoginResponse{}, nil
}
func (m *mockService) Logout(ctx context.Context, refreshToken string) error { return nil }
func (m *mockService) RefreshToken(ctx context.Context, refreshToken string) (dto.LoginResponse, error) {
	return dto.LoginResponse{}, nil
}
func (m *mockService) GetUserByID(ctx context.Context, userID int) (dto.PublicUser, error) {
	return dto.PublicUser{}, nil
}
func (m *mockService) UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) error {
	return nil
}
func (m *mockService) ChangePassword(ctx context.Context, userID int, req dto.ChangePasswordRequest) error {
	m.ChangePasswordCalled = true
	m.LastChangePasswordUserID = userID
	m.LastChangePasswordReq = req
	return nil
}
func (m *mockService) ChangePasswordAdmin(ctx context.Context, userID int, newPassword string) error {
	m.ChangePasswordAdminCalled = true
	m.LastChangePasswordUserID = userID
	m.LastAdminNewPassword = newPassword
	return nil
}
func (m *mockService) DeactivateAccount(ctx context.Context, userID int) error { return nil }
func (m *mockService) ActivateAccount(ctx context.Context, userID int) error   { return nil }
func (m *mockService) ListUsers(ctx context.Context, req dto.ListUsersRequest) (dto.ListUsersResponse, error) {
	return dto.ListUsersResponse{}, nil
}
func (m *mockService) DeleteUser(ctx context.Context, userID int) (int, error) { return 0, nil }

func TestHandleChangePassword_AdminPath(t *testing.T) {
	m := &mockService{}
	h := NewUserHandler(m)

	body := map[string]string{"new_password": "adminnew"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/users/5/password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// set chi route param id=5
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "5")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	// set admin role in context
	ctx := context.WithValue(req.Context(), ctxKeyUserRole, "admin")
	ctx = context.WithValue(ctx, ctxKeyUserID, 3) // admin id
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.handleChangePassword(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rr.Code, rr.Body.String())
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
	h := NewUserHandler(m)

	body := map[string]string{"current_password": "old", "new_password": "new"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/users/7/password", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "7")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	// set owner id same as target
	ctx := context.WithValue(req.Context(), ctxKeyUserRole, "client")
	ctx = context.WithValue(ctx, ctxKeyUserID, 7)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.handleChangePassword(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !m.ChangePasswordCalled {
		t.Fatalf("expected ChangePassword called")
	}
	if m.LastChangePasswordUserID != 7 {
		t.Fatalf("expected userID 7, got %d", m.LastChangePasswordUserID)
	}
}
