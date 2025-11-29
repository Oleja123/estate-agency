package propertyhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	imagedto "github.com/Oleja123/estate-agency/internal/application/image/dto"
	imagedomain "github.com/Oleja123/estate-agency/internal/domain/image"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	favdto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	favdomain "github.com/Oleja123/estate-agency/internal/domain/favorite"
	domain "github.com/Oleja123/estate-agency/internal/domain/property"
	auth "github.com/Oleja123/estate-agency/internal/handler/auth"
	"github.com/go-chi/chi/v5"
)

// mockService implements the property service interface used by handlers.
type mockService struct {
	CreateCalled bool
	CreateFunc   func(ctx context.Context, userID int, req dto.CreatePropertyRequest) (domain.Property, error)

	GetByIDCalled bool
	GetByIDFunc   func(ctx context.Context, id int) (domain.Property, error)

	UpdateCalled bool
	UpdateFunc   func(ctx context.Context, req dto.UpdatePropertyRequest) error

	ListCalled bool
	ListFunc   func(ctx context.Context, req dto.ListPropertiesRequest) (dto.ListPropertiesResponse, error)

	DeleteCalled bool
	DeleteFunc   func(ctx context.Context, id int) (int, error)
	// ToggleFavoriteFunc allows tests to intercept ToggleFavorite calls.
	ToggleFavoriteFunc func(ctx context.Context, userID int, propertyID int) (bool, favdomain.Favorite, error)
}

func (m *mockService) Create(ctx context.Context, userID int, req dto.CreatePropertyRequest) (domain.Property, error) {
	m.CreateCalled = true
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, userID, req)
	}
	return domain.Property{ID: 1, Title: req.Title, CreatedBy: userID}, nil
}
func (m *mockService) GetByID(ctx context.Context, id int) (domain.Property, error) {
	m.GetByIDCalled = true
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return domain.Property{ID: id, Title: "t", CreatedBy: 3}, nil
}
func (m *mockService) Update(ctx context.Context, req dto.UpdatePropertyRequest) error {
	m.UpdateCalled = true
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, req)
	}
	return nil
}
func (m *mockService) List(ctx context.Context, req dto.ListPropertiesRequest) (dto.ListPropertiesResponse, error) {
	m.ListCalled = true
	if m.ListFunc != nil {
		return m.ListFunc(ctx, req)
	}
	return dto.ListPropertiesResponse{Properties: []dto.PropertyDTO{{ID: 1, Title: "a"}}, Total: 1}, nil
}
func (m *mockService) Delete(ctx context.Context, id int) (int, error) {
	m.DeleteCalled = true
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return id, nil
}

func (m *mockService) ToggleFavorite(ctx context.Context, userID int, propertyID int) (bool, favdomain.Favorite, error) {
	if m.ToggleFavoriteFunc != nil {
		return m.ToggleFavoriteFunc(ctx, userID, propertyID)
	}
	return false, favdomain.Favorite{}, nil
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// mockImageService implements the image service used by the property handler for tests.
type mockImageService struct {
	CreateManyCalled bool
	CreateManyFunc   func(ctx context.Context, req imagedto.CreateImagesRequest) ([]imagedomain.PropertyImage, error)
}

func (m *mockImageService) Create(ctx context.Context, req imagedto.CreateImageRequest) (imagedomain.PropertyImage, error) {
	return imagedomain.PropertyImage{}, nil
}
func (m *mockImageService) CreateMany(ctx context.Context, req imagedto.CreateImagesRequest) ([]imagedomain.PropertyImage, error) {
	m.CreateManyCalled = true
	if m.CreateManyFunc != nil {
		return m.CreateManyFunc(ctx, req)
	}
	return []imagedomain.PropertyImage{{ID: 1, PropertyID: req.PropertyID, Path: "p/1.jpg"}}, nil
}
func (m *mockImageService) GetByID(ctx context.Context, id int) (imagedomain.PropertyImage, error) {
	return imagedomain.PropertyImage{}, nil
}
func (m *mockImageService) ListByProperty(ctx context.Context, propertyID int) ([]imagedto.ImageDTO, error) {
	return nil, nil
}
func (m *mockImageService) Delete(ctx context.Context, id int) (int, error) {
	return 0, nil
}

// mockFavorite is a small favorites service mock used by property handler tests.
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

func TestHandleCreate_CallsServiceAndReturnsCreated(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	body := map[string]interface{}{"title": "My prop"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/properties", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	req = req.WithContext(auth.ContextWithUser(req.Context(), 1, "admin"))
	h.handleCreate(rr, req)

	if !m.CreateCalled {
		t.Fatalf("expected Create to be called")
	}
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	var got domain.Property
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Title != "My prop" {
		t.Fatalf("unexpected title: %s", got.Title)
	}
}

func TestHandleUpdateImages_CallsServiceAndReturnsNoContent(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	imgMock := &mockImageService{}
	h := NewPropertyHandler(m, logger, &mockFavorite{}, imgMock)

	// build multipart body with one file
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("files", "a.jpg")
	fw.Write([]byte("hello"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPut, "/properties/3/images", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "3")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	h.handleUpdateImages(rr, req)
	if !imgMock.CreateManyCalled {
		t.Fatalf("expected CreateMany to be called on image service")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateImages_TooManyFiles_Returns400(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	imgMock := &mockImageService{}
	h := NewPropertyHandler(m, logger, &mockFavorite{}, imgMock)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for i := 0; i < 11; i++ {
		fw, _ := mw.CreateFormFile("files", fmt.Sprintf("file%d.jpg", i))
		fw.Write([]byte("x"))
	}
	mw.Close()

	req := httptest.NewRequest(http.MethodPut, "/properties/3/images", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "3")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	h.handleUpdateImages(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for too many files on update, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreate_InvalidJSON_Returns400(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/properties", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	req = req.WithContext(auth.ContextWithUser(req.Context(), 1, "admin"))
	h.handleCreate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d", rr.Code)
	}
}

func TestHandleCreate_Middleware_AdminAllowed(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Post("/", http.HandlerFunc(h.handleCreate))

	body := map[string]interface{}{"title": "My prop"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// set admin in context
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
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Post("/", http.HandlerFunc(h.handleCreate))

	body := map[string]interface{}{"title": "My prop"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	// set non-admin in context
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
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/properties/2", nil)
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

// Service error mapping tests
func TestHandleCreate_ServiceInvalidInput_Returns400(t *testing.T) {
	m := &mockService{}
	m.CreateFunc = func(ctx context.Context, userID int, req dto.CreatePropertyRequest) (domain.Property, error) {
		return domain.Property{}, apperrors.NewErrInvalidInput("title", req.Title, "invalid")
	}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	body := map[string]interface{}{"title": "", "type_id": 1}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/properties", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	req = req.WithContext(auth.ContextWithUser(req.Context(), 1, "admin"))
	h.handleCreate(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid input, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleCreate_ServiceAlreadyExists_Returns409(t *testing.T) {
	m := &mockService{}
	m.CreateFunc = func(ctx context.Context, userID int, req dto.CreatePropertyRequest) (domain.Property, error) {
		return domain.Property{}, apperrors.NewErrAlreadyExists("property", "title", req.Title)
	}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	body := map[string]interface{}{"title": "dup", "type_id": 1}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/properties", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	req = req.WithContext(auth.ContextWithUser(req.Context(), 1, "admin"))
	h.handleCreate(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 for already exists, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleGet_NotFound_Returns404(t *testing.T) {
	m := &mockService{}
	m.GetByIDFunc = func(ctx context.Context, id int) (domain.Property, error) {
		return domain.Property{}, apperrors.NewErrNotFound("property", id)
	}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/properties/99", nil)
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
	m.UpdateFunc = func(ctx context.Context, req dto.UpdatePropertyRequest) error {
		return apperrors.NewErrNotFound("property", req.ID)
	}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	req := httptest.NewRequest(http.MethodPatch, "/3", bytes.NewReader([]byte(`{"title":"x"}`)))
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
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

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

func TestHandleUpdate_CallsServiceAndReturnsNoContent(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	req := httptest.NewRequest(http.MethodPatch, "/properties/3", bytes.NewReader([]byte(`{"title":"x"}`)))
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "3")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	h.handleUpdate(rr, req)
	if !m.UpdateCalled {
		t.Fatalf("expected Update called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdate_Middleware_AdminAllowed(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Patch("/{id}", http.HandlerFunc(h.handleUpdate))

	req := httptest.NewRequest(http.MethodPatch, "/3", bytes.NewReader([]byte(`{"title":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), 2, "admin"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for admin update, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdate_Middleware_NonAdminForbidden(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	r := chi.NewRouter()
	r.With(auth.RequireAdminMiddleware()).Patch("/{id}", http.HandlerFunc(h.handleUpdate))

	req := httptest.NewRequest(http.MethodPatch, "/3", bytes.NewReader([]byte(`{"title":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.ContextWithUser(req.Context(), 2, "client"))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin update, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDelete_CallsServiceAndReturnsOK(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/properties/4", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", "4")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
	rr := httptest.NewRecorder()

	h.handleDelete(rr, req)
	if !m.DeleteCalled {
		t.Fatalf("expected Delete called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleDelete_Middleware_AdminAllowed(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, &mockFavorite{}, nil)

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

func TestHandleDelete_Middleware_NonAdminForbidden(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, nil, nil)

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

func TestHandleList_CallsServiceAndReturnsList(t *testing.T) {
	m := &mockService{}
	logger := newLogger()
	h := NewPropertyHandler(m, logger, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/properties", nil)
	rr := httptest.NewRecorder()

	h.handleList(rr, req)
	if !m.ListCalled {
		t.Fatalf("expected List called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}
