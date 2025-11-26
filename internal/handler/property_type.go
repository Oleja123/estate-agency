package handler

import (
	"net/http"
	"strconv"
	"strings"

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

func (h *PropertyTypeHandler) Register(mux *http.ServeMux, prefix string) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	mux.HandleFunc(prefix, h.handlePropertyTypes)
}

func (h *PropertyTypeHandler) handlePropertyTypes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
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
	case http.MethodGet:
		path := strings.TrimPrefix(r.URL.Path, "/property_types/")
		if path == "" || path == "/" {
			writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "list not implemented"})
			return
		}
		id, err := strconv.Atoi(strings.Trim(path, "/"))
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
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
