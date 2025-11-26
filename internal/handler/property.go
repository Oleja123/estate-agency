package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	propertysvc "github.com/Oleja123/estate-agency/internal/application/property"
	dto "github.com/Oleja123/estate-agency/internal/application/property/dto"
)

type PropertyHandler struct {
	svc propertysvc.Service
}

func NewPropertyHandler(s propertysvc.Service) *PropertyHandler {
	return &PropertyHandler{svc: s}
}

func (h *PropertyHandler) Register(r chi.Router, prefix string) {
	if prefix == "" {
		prefix = "/properties"
	}
	r.Route(prefix, func(r chi.Router) {
		r.Post("/", h.handleCreate)
		r.Get("/", h.handleList)
		r.Get("/{id}", h.handleGet)
		r.Put("/{id}", h.handleUpdate)
		r.Delete("/{id}", h.handleDelete)
	})
}

func (h *PropertyHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePropertyRequest
	if err := decodeJSON(r, &req); err != nil {
		code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		writeJSON(w, code, body)
		return
	}
	p, err := h.svc.Create(r.Context(), req)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *PropertyHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	p, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, p)
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
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *PropertyHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req dto.UpdatePropertyRequest
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

func (h *PropertyHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
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
