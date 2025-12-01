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
	maxImagesPerRequest = 10

	multipartMaxMemory = 32 << 20
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

	if authMw == nil {
		authMw = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
			})
		}
	}

	r.Route(prefix, func(r chi.Router) {

		r.Use(authMw)

		r.Get("/", h.handleList)
		r.Get("/{id}", h.handleGet)

		r.With(auth.RequireAdminMiddleware()).Post("/", h.handleCreate)
		r.With(auth.RequireAdminMiddleware()).Patch("/{id}", h.handleUpdate)
		r.With(auth.RequireAdminMiddleware()).Delete("/{id}", h.handleDelete)

		if h.favSvc != nil {

			r.Post("/{id}/favorites", h.handleToggleFavorite)
		}

		if h.imgSvc != nil {

			r.Get("/{id}/images", h.handleListImages)
			r.With(auth.RequireAdminMiddleware()).Put("/{id}/images", h.handleUpdateImages)
		}
	})
}

func (h *PropertyHandler) handleToggleFavorite(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
		return
	}

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

func (h *PropertyHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePropertyRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "некорректный JSON"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	uid, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "требуется авторизация"})
		return
	}
	p, err := h.svc.Create(r.Context(), uid, req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}

	handlerutils.WriteJSON(w, http.StatusCreated, propertysvc.MapProperty(p))
}

func (h *PropertyHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}
	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}

	handlerutils.WriteJSON(w, http.StatusOK, propertysvc.MapProperty(p))
}

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

	var f propdomain.Filter

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

func (h *PropertyHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}
	var req dto.UpdatePropertyRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "некорректный JSON"))
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

func (h *PropertyHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
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

func (h *PropertyHandler) handleUpdateImages(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}
	if h.imgSvc == nil {
		handlerutils.WriteJSON(w, http.StatusNotImplemented, map[string]string{"error": "загрузка изображений не поддерживается"})
		return
	}
	if err := r.ParseMultipartForm(multipartMaxMemory); err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректная multipart форма"})
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "файлы не предоставлены"})
		return
	}
	if len(files) > maxImagesPerRequest {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "слишком много файлов; допускается максимум 10"})
		return
	}
	var req imagedto.CreateImagesRequest
	req.PropertyID = pid
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "не удалось прочитать файл"})
			return
		}
		data, err := io.ReadAll(f)
		_ = f.Close()
		if err != nil {
			handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "не удалось прочитать файл"})
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

func (h *PropertyHandler) handleListImages(w http.ResponseWriter, r *http.Request) {
	pidStr := chi.URLParam(r, "id")
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}
	if h.imgSvc == nil {
		handlerutils.WriteJSON(w, http.StatusNotImplemented, map[string]string{"error": "загрузка изображений не поддерживается"})
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
