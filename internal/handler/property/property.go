package propertyhandler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	propertysvc "github.com/Oleja123/estate-agency/internal/application/property"
	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
	"github.com/Oleja123/estate-agency/internal/handler/auth"
	handlerutils "github.com/Oleja123/estate-agency/internal/handler/utils"
)

type PropertyHandler struct {
	svc propertysvc.Service
	lg  *slog.Logger
}

func NewPropertyHandler(s propertysvc.Service, l *slog.Logger) *PropertyHandler {
	return &PropertyHandler{svc: s, lg: l}
}

func (h *PropertyHandler) Register(r chi.Router, prefix string, authMw func(next http.Handler) http.Handler) {
	if prefix == "" {
		prefix = "/properties"
	}
	r.Route(prefix, func(r chi.Router) {
		// public reads
		r.Get("/", h.handleList)
		r.Get("/{id}", h.handleGet)
		// create/update/delete require auth and admin role
		if authMw != nil {
			r.Group(func(r chi.Router) {
				r.Use(authMw)
				r.With(auth.RequireAdminMiddleware()).Post("/", h.handleCreate)
				r.With(auth.RequireAdminMiddleware()).Put("/{id}", h.handleUpdate)
				r.With(auth.RequireAdminMiddleware()).Delete("/{id}", h.handleDelete)
			})
		} else {
			// no auth middleware available in this environment — leave handlers attached
			r.Post("/", h.handleCreate)
			r.Put("/{id}", h.handleUpdate)
			r.Delete("/{id}", h.handleDelete)
		}
	})
}

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
