package propertytypehandler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	auth "github.com/Oleja123/estate-agency/internal/handler/auth"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	typesvc "github.com/Oleja123/estate-agency/internal/application/property_type"
	dto "github.com/Oleja123/estate-agency/internal/application/property_type/dto"
	handlerutils "github.com/Oleja123/estate-agency/internal/handler/utils"
)

type PropertyTypeHandler struct {
	svc typesvc.Service
	lg  *slog.Logger
}

func NewPropertyTypeHandler(s typesvc.Service, l *slog.Logger) *PropertyTypeHandler {
	return &PropertyTypeHandler{svc: s, lg: l}
}

func (h *PropertyTypeHandler) Register(r chi.Router, prefix string, authMw func(next http.Handler) http.Handler) {
	if prefix == "" {
		prefix = "/property_types"
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
	})
}

func (h *PropertyTypeHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePropertyTypeRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "некорректный JSON"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	t, err := h.svc.Create(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusCreated, t)
}

func (h *PropertyTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}
	t, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, t)
}

func (h *PropertyTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
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
	req := dto.ListPropertyTypesRequest{Limit: limit, Offset: offset}
	res, err := h.svc.List(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, res)
}

func (h *PropertyTypeHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}
	var req dto.UpdatePropertyTypeRequest
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

func (h *PropertyTypeHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
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
