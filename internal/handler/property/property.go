package propertyhandler

import (
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	favoritesvc "github.com/Oleja123/estate-agency/internal/application/favorite"

	imagesvc "github.com/Oleja123/estate-agency/internal/application/image"
	imagedto "github.com/Oleja123/estate-agency/internal/application/image/dto"
	propertysvc "github.com/Oleja123/estate-agency/internal/application/property"
	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	propdomain "github.com/Oleja123/estate-agency/internal/domain/property"
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
		r.With(auth.RequireAdminMiddleware()).Patch("/{id}", h.handleUpdate)
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
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
		return
	}
	p, err := h.svc.Create(r.Context(), uid, req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	// map domain.Property to API DTO to avoid exposing internal fields like created_by
	handlerutils.WriteJSON(w, http.StatusCreated, propertysvc.MapProperty(p))
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
	// map domain.Property to API DTO using centralized mapper
	handlerutils.WriteJSON(w, http.StatusOK, propertysvc.MapProperty(p))
}

// @Security BearerAuth
// @Summary List properties
// @Description List properties with pagination and filters
// @Tags properties
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param ids query string false "Comma-separated property IDs"
// @Param type_ids query string false "Comma-separated property type IDs"
// @Param transaction_type query string false "transaction type (sale|rent)"
// @Param city query string false "City name"
// @Param property_status query string false "Property status (active|sold|rented|inactive)"
// @Param created_by query int false "Creator user ID"
// @Param min_price query number false "Minimum price"
// @Param max_price query number false "Maximum price"
// @Param min_area query number false "Minimum area"
// @Param max_area query number false "Maximum area"
// @Param search query string false "Full-text search query"
// @Param latitude query number false "Latitude for geo filter"
// @Param longitude query number false "Longitude for geo filter"
// @Param radius_km query number false "Radius in kilometers for geo filter"
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
	// build filter from query params
	var f propdomain.Filter

	// helper to parse comma-separated ints
	parseInts := func(s string) []int {
		if s == "" {
			return nil
		}
		var out []int
		for _, part := range strings.Split(s, ",") {
			p := strings.TrimSpace(part)
			if p == "" {
				continue
			}
			if v, err := strconv.Atoi(p); err == nil {
				out = append(out, v)
			}
		}
		return out
	}

	if ids := q.Get("ids"); ids != "" {
		f.IDs = parseInts(ids)
	}
	if tids := q.Get("type_ids"); tids != "" {
		f.TypeIDs = parseInts(tids)
	} else if tid := q.Get("type_id"); tid != "" {
		f.TypeIDs = parseInts(tid)
	}
	if tt := q.Get("transaction_type"); tt != "" {
		f.TransactionType = propdomain.TransactionType(tt)
	}
	if city := q.Get("city"); city != "" {
		f.City = city
	}
	if ps := q.Get("property_status"); ps != "" {
		f.PropertyStatus = propdomain.PropertyStatus(ps)
	}
	if cb := q.Get("created_by"); cb != "" {
		if v, err := strconv.Atoi(cb); err == nil {
			f.CreatedBy = v
		}
	}
	if mp := q.Get("min_price"); mp != "" {
		if v, err := strconv.ParseFloat(mp, 64); err == nil {
			f.MinPrice = v
		}
	}
	if xp := q.Get("max_price"); xp != "" {
		if v, err := strconv.ParseFloat(xp, 64); err == nil {
			f.MaxPrice = v
		}
	}
	if ma := q.Get("min_area"); ma != "" {
		if v, err := strconv.ParseFloat(ma, 64); err == nil {
			f.MinArea = v
		}
	}
	if xa := q.Get("max_area"); xa != "" {
		if v, err := strconv.ParseFloat(xa, 64); err == nil {
			f.MaxArea = v
		}
	}
	if s := q.Get("search"); s != "" {
		f.Search = s
	}
	if lat := q.Get("latitude"); lat != "" {
		if v, err := strconv.ParseFloat(lat, 64); err == nil {
			f.Latitude = v
		}
	}
	if lon := q.Get("longitude"); lon != "" {
		if v, err := strconv.ParseFloat(lon, 64); err == nil {
			f.Longitude = v
		}
	}
	if rk := q.Get("radius_km"); rk != "" {
		if v, err := strconv.ParseFloat(rk, 64); err == nil {
			f.RadiusKm = v
		}
	}

	req := dto.ListPropertiesRequest{Filter: f, Limit: limit, Offset: offset}
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
// @Description Partially update an existing property
// @Tags properties
// @Accept json
// @Produce json
// @Param id path int true "Property ID"
// @Param body body UpdatePropertyDoc true "Property"
// @Success 204
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /properties/{id} [patch]
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

// @Param id path int true "Property ID"
// @Param files formData []file true "Files"
// @Success 201 {array} ImageDTODoc
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
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
