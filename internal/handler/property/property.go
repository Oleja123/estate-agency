package propertyhandler

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	favoritesvc "github.com/Oleja123/estate-agency/internal/application/favorite"

	imagesvc "github.com/Oleja123/estate-agency/internal/application/image"
	imagedto "github.com/Oleja123/estate-agency/internal/application/image/dto"
	propertysvc "github.com/Oleja123/estate-agency/internal/application/property"
	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	"github.com/Oleja123/estate-agency/internal/handler/auth"
	handlerutils "github.com/Oleja123/estate-agency/internal/handler/utils"
)

const (
	// maxImagesPerRequest is the maximum number of files accepted in a single
	// multipart upload for property images.
	maxImagesPerRequest = 10
	// multipartMaxMemory is the max memory passed to ParseMultipartForm.
	multipartMaxMemory = 32 << 20 // 32 MiB
)

type PropertyHandler struct {
	svc    propertysvc.Service
	lg     *slog.Logger
	favSvc favoritesvc.Service
	imgSvc imagesvc.Service
}

func NewPropertyHandler(s propertysvc.Service, l *slog.Logger, fav favoritesvc.Service, img imagesvc.Service) *PropertyHandler {
	return &PropertyHandler{svc: s, lg: l, favSvc: fav, imgSvc: img}
}

func (h *PropertyHandler) Register(r chi.Router, prefix string, authMw func(next http.Handler) http.Handler) {
	if prefix == "" {
		prefix = "/properties"
	}
	// Enforce auth for all property endpoints. If auth middleware is not
	// provided, insert a denying middleware so endpoints are not exposed.
	if authMw == nil {
		authMw = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
			})
		}
	}

	r.Route(prefix, func(r chi.Router) {
		// Require authentication for all routes under /properties
		r.Use(authMw)

		// reads now require authentication as well
		r.Get("/", h.handleList)
		r.Get("/{id}", h.handleGet)

		// create/update/delete require admin role
		r.With(auth.RequireAdminMiddleware()).Post("/", h.handleCreate)
		r.With(auth.RequireAdminMiddleware()).Put("/{id}", h.handleUpdate)
		r.With(auth.RequireAdminMiddleware()).Delete("/{id}", h.handleDelete)

		// allow authenticated users to toggle favorite for a property
		if h.favSvc != nil {
			// toggle favorite for current user: POST /{id}/favorites
			r.Post("/{id}/favorites", h.handleToggleFavorite)
		}

		// images upload/update endpoints (admin-protected). Attach only if image service present.
		if h.imgSvc != nil {
			// allow authenticated users to list images for a property
			r.Get("/{id}/images", h.handleListImages)
			r.With(auth.RequireAdminMiddleware()).Post("/{id}/images", h.handleCreateImages)
			r.With(auth.RequireAdminMiddleware()).Put("/{id}/images", h.handleUpdateImages)
		}
	})
}

// constructor-only injection enforced; SetFavoriteService removed

// handleToggleFavorite toggles favorite for the authenticated user and the
// given property id. If the property was not favorited, it creates one and
// returns 201 with the created object. If it was favorited already, it
// deletes it and returns 204 No Content.
// @Security BearerAuth
// @Summary Toggle favorite for property
// @Description Toggle favorite for the authenticated user and given property ID. Returns 201 with created favorite or 204 when removed.
// @Tags properties
// @Accept json
// @Produce json
// @Param id path int true "Property ID"
// @Success 201 {object} FavoriteDTODoc
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /properties/{id}/favorites [post]
func (h *PropertyHandler) handleToggleFavorite(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
		return
	}

	// Delegate favorite toggle to property service: it will call favorites service.
	created, fav, err := h.svc.ToggleFavorite(r.Context(), uid, pid)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	if created {
		handlerutils.WriteJSON(w, http.StatusCreated, fav)
		return
	}
	handlerutils.WriteJSON(w, http.StatusNoContent, nil)
}

// @Security BearerAuth
// @Summary Create property
// @Description Create a new property
// @Tags properties
// @Accept json
// @Produce json
// @Param body body PropertyCreateDoc true "Property"
// @Success 201 {object} PropertyDTODoc
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /properties [post]
func (h *PropertyHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePropertyRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	p, err := h.svc.Create(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusCreated, p)
}

// @Security BearerAuth
// @Summary Get property
// @Description Get property by ID
// @Tags properties
// @Accept json
// @Produce json
// @Param id path int true "Property ID"
// @Success 200 {object} PropertyDTODoc
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /properties/{id} [get]
func (h *PropertyHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, p)
}

// @Security BearerAuth
// @Summary List properties
// @Description List properties with pagination
// @Tags properties
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} ListPropertiesResponseDoc
// @Router /properties [get]
func (h *PropertyHandler) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 20
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 500 {
		limit = 500
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	req := dto.ListPropertiesRequest{Limit: limit, Offset: offset}
	res, err := h.svc.List(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, res)
}

// @Security BearerAuth
// @Summary Update property
// @Description Update an existing property
// @Tags properties
// @Accept json
// @Produce json
// @Param id path int true "Property ID"
// @Param body body UpdatePropertyDoc true "Property"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /properties/{id} [put]
func (h *PropertyHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req dto.UpdatePropertyRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	req.ID = id
	if err := h.svc.Update(r.Context(), req); err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusNoContent, nil)
}

// @Security BearerAuth
// @Summary Delete property
// @Description Delete property by ID
// @Tags properties
// @Accept json
// @Produce json
// @Param id path int true "Property ID"
// @Success 200 {object} map[string]int
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /properties/{id} [delete]
func (h *PropertyHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	deletedID, err := h.svc.Delete(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, map[string]int{"deleted_id": deletedID})
}

// handleCreateImages handles multipart file upload (field name "files") and creates property images.
// @Security BearerAuth
// @Summary Upload images for a property
// @Description Upload up to 10 image files for the given property. Field name: files (multipart/form-data).
// @Tags properties
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Property ID"
// @Param files formData []file true "Files"
// @Success 201 {array} ImageDTODoc
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 415 {object} map[string]string
// @Router /properties/{id}/images [post]
func (h *PropertyHandler) handleCreateImages(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if h.imgSvc == nil {
		handlerutils.WriteJSON(w, http.StatusNotImplemented, map[string]string{"error": "images not supported"})
		return
	}
	if err := r.ParseMultipartForm(multipartMaxMemory); err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "no files provided"})
		return
	}
	if len(files) > maxImagesPerRequest {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "too many files; maximum 10 allowed"})
		return
	}
	var req imagedto.CreateImagesRequest
	req.PropertyID = pid
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read file"})
			return
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read file"})
			return
		}
		req.Files = append(req.Files, imagedto.ImageFile{Filename: fh.Filename, Data: data})
	}
	imgs, err := h.imgSvc.CreateMany(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusCreated, imgs)
}

// @Param id path int true "Property ID"
// @Param files formData []file true "Files"
// @Success 201 {array} ImageDTODoc
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 415 {object} map[string]string
// @Router /properties/{id}/images [post]

// handleUpdateImages replaces existing images for the property with provided files.
// @Security BearerAuth
// @Summary Replace images for a property
// @Description Replace existing images for the given property with provided files (up to 10). Field name: files (multipart/form-data).
// @Tags properties
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Property ID"
// @Param files formData file true "Files"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 415 {object} map[string]string
// @Router /properties/{id}/images [put]
func (h *PropertyHandler) handleUpdateImages(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if h.imgSvc == nil {
		handlerutils.WriteJSON(w, http.StatusNotImplemented, map[string]string{"error": "images not supported"})
		return
	}
	if err := r.ParseMultipartForm(multipartMaxMemory); err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "no files provided"})
		return
	}
	if len(files) > maxImagesPerRequest {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "too many files; maximum 10 allowed"})
		return
	}
	var req imagedto.CreateImagesRequest
	req.PropertyID = pid
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read file"})
			return
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot read file"})
			return
		}
		req.Files = append(req.Files, imagedto.ImageFile{Filename: fh.Filename, Data: data})
	}
	_, err = h.imgSvc.CreateMany(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusNoContent, nil)
}

// @Security BearerAuth
// @Summary List images for a property
// @Description Get list of images for the given property ID. Requires authentication.
// @Tags properties
// @Accept json
// @Produce json
// @Param id path int true "Property ID"
// @Success 200 {array} ImageDTODoc
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /properties/{id}/images [get]
func (h *PropertyHandler) handleListImages(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if h.imgSvc == nil {
		handlerutils.WriteJSON(w, http.StatusNotImplemented, map[string]string{"error": "images not supported"})
		return
	}
	imgs, err := h.imgSvc.ListByProperty(r.Context(), pid)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, imgs)
}
