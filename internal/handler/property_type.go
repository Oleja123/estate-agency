package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	typesvc "github.com/Oleja123/estate-agency/internal/application/property_type"
	dto "github.com/Oleja123/estate-agency/internal/application/property_type/dto"
)

type PropertyTypeHandler struct {
	svc typesvc.Service
}

func NewPropertyTypeHandler(s typesvc.Service) *PropertyTypeHandler {
	return &PropertyTypeHandler{svc: s}
}

func (h *PropertyTypeHandler) Register(r chi.Router, prefix string) {
	if prefix == "" {
		prefix = "/property_types"
	}
	r.Route(prefix, func(r chi.Router) {
		r.Post("/", h.handleCreate)
		r.Get("/", h.handleList)
		r.Get("/{id}", h.handleGet)
		r.Put("/{id}", h.handleUpdate)
		r.Delete("/{id}", h.handleDelete)
	})
}

func (h *PropertyTypeHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePropertyTypeRequest
	if err := decodeJSON(r, &req); err != nil {
		code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		writeJSON(w, code, body)
		return
	}
	t, err := h.svc.Create(r.Context(), req)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *PropertyTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	t, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, t)
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
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *PropertyTypeHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req dto.UpdatePropertyTypeRequest
	if err := decodeJSON(r, &req); err != nil {
		code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		writeJSON(w, code, body)
		return
	}
	req.ID = id
	if err := h.svc.Update(r.Context(), req); err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (h *PropertyTypeHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	deletedID, err := h.svc.Delete(r.Context(), id)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"deleted_id": deletedID})
}
