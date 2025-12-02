package propertytypehandler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/property_type/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/property_type"
	auth "github.com/Oleja123/estate-agency/internal/handler/auth"
	"github.com/go-chi/chi/v5"
)

type mockService struct {
	CreateCalled bool
	CreateFunc   func(ctx context.Context, req dto.CreatePropertyTypeRequest) (domain.PropertyType, error)

	GetByIDCalled bool
	GetByIDFunc   func(ctx context.Context, id int) (domain.PropertyType, error)

	UpdateCalled bool
	UpdateFunc   func(ctx context.Context, req dto.UpdatePropertyTypeRequest) (domain.PropertyType, error)

	ListCalled bool
	ListFunc   func(ctx context.Context, req dto.ListPropertyTypesRequest) (dto.ListPropertyTypesResponse, error)

	DeleteCalled bool
	DeleteFunc   func(ctx context.Context, id int) (int, error)
}

func (m *mockService) Create(ctx context.Context, req dto.CreatePropertyTypeRequest) (domain.PropertyType, error) {
	m.CreateCalled = true
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return domain.PropertyType{Id: 1, Name: req.Name}, nil
}
func (m *mockService) GetByID(ctx context.Context, id int) (domain.PropertyType, error) {
	m.GetByIDCalled = true
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return domain.PropertyType{Id: id, Name: "t"}, nil
}
func (m *mockService) Update(ctx context.Context, req dto.UpdatePropertyTypeRequest) (domain.PropertyType, error) {
	m.UpdateCalled = true
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, req)
	}
	return domain.PropertyType{Id: req.ID, Name: req.Name}, nil
}
func (m *mockService) List(ctx context.Context, req dto.ListPropertyTypesRequest) (dto.ListPropertyTypesResponse, error) {
	m.ListCalled = true
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return dto.ListPropertyTypesResponse{Types: []domain.PropertyType{{Id: 1, Name: "a"}}, Total: 1}, nil
}
func (m *mockService) Delete(ctx context.Context, id int) (int, error) {
	m.DeleteCalled = true
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return id, nil
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestHandleCreate_CallsServiceAndReturnsCreated(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	body := map[string]interface{}{"name": "MyType"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/property_types", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.handleCreate(rr, req)

	if !m.CreateCalled {
		t.Fatalf("expected Create to be called")
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got domain.PropertyType
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "MyType" {
		t.Fatalf("unexpected name: %s", got.Name)
	}
}

func TestHandleCreate_Middleware_AdminAllowed(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Post("/", http.HandlerFunc(h.handleCreate))

	body := map[string]interface{}{"name": "MyType"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), 10, "admin"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreate_Middleware_NonAdminForbidden(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Post("/", http.HandlerFunc(h.handleCreate))

	body := map[string]interface{}{"name": "MyType"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), 11, "client"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGet_CallsServiceAndReturns(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	req := httptest.NewRequest(http.MethodGet, "/property_types/2", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "2")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	h.handleGet(rr, req)
	if !m.GetByIDCalled {
		t.Fatalf("expected GetByID called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestHandleList_CallsServiceAndReturnsList(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	req := httptest.NewRequest(http.MethodGet, "/property_types", nil)
	rr := httptest.NewRecorder()

	h.handleList(rr, req)
	if !m.ListCalled {
		t.Fatalf("expected List called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestHandleUpdate_Middleware_AdminAllowed(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Patch("/{id}", http.HandlerFunc(h.handleUpdate))

	req := httptest.NewRequest(http.MethodPatch, "/3", bytes.NewReader([]byte(`{"name":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), 2, "admin"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin update, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDelete_Middleware_AdminAllowed(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Delete("/{id}", http.HandlerFunc(h.handleDelete))

	req := httptest.NewRequest(http.MethodDelete, "/4", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), 1, "admin"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin delete, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdate_Middleware_NonAdminForbidden(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Patch("/{id}", http.HandlerFunc(h.handleUpdate))

	req := httptest.NewRequest(http.MethodPatch, "/3", bytes.NewReader([]byte(`{"name":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), 2, "client"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin update, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDelete_Middleware_NonAdminForbidden(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Delete("/{id}", http.HandlerFunc(h.handleDelete))

	req := httptest.NewRequest(http.MethodDelete, "/4", nil)
	req = req.WithContext(auth.ContextWithUser(req.Context(), 1, "client"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin delete, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreate_ServiceInvalidInput_Returns400(t *testing.T) {
	m := &mockService{}
	m.CreateFunc = func(ctx context.Context, req dto.CreatePropertyTypeRequest) (domain.PropertyType, error) {
		return domain.PropertyType{}, apperrors.NewErrInvalidInput("name", req.Name, "invalid")
	}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	body := map[string]interface{}{"name": ""}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/property_types", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.handleCreate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid input, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreate_ServiceAlreadyExists_Returns409(t *testing.T) {
	m := &mockService{}
	m.CreateFunc = func(ctx context.Context, req dto.CreatePropertyTypeRequest) (domain.PropertyType, error) {
		return domain.PropertyType{}, apperrors.NewErrAlreadyExists("property_type", "name", req.Name)
	}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	body := map[string]interface{}{"name": "dup"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/property_types", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.handleCreate(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 for already exists, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGet_NotFound_Returns404(t *testing.T) {
	m := &mockService{}
	m.GetByIDFunc = func(ctx context.Context, id int) (domain.PropertyType, error) {
		return domain.PropertyType{}, apperrors.NewErrNotFound("property_type", id)
	}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	req := httptest.NewRequest(http.MethodGet, "/property_types/99", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "99")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	h.handleGet(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for not found, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdate_NotFound_Returns404(t *testing.T) {
	m := &mockService{}
	m.UpdateFunc = func(ctx context.Context, req dto.UpdatePropertyTypeRequest) (domain.PropertyType, error) {
		return domain.PropertyType{}, apperrors.NewErrNotFound("property_type", req.ID)
	}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	req := httptest.NewRequest(http.MethodPatch, "/3", bytes.NewReader([]byte(`{"name":"x"}`)))
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "3")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	h.handleUpdate(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for update not found, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDelete_InternalError_Returns500(t *testing.T) {
	m := &mockService{}
	m.DeleteFunc = func(ctx context.Context, id int) (int, error) {
		return 0, apperrors.NewErrInternal("db failure")
	}
	logger := newLogger()
	h := NewPropertyTypeHandler(m, logger)

	req := httptest.NewRequest(http.MethodDelete, "/4", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "4")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	h.handleDelete(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for internal error, got %d body=%s", rr.Code, rr.Body.String())
	}
}
