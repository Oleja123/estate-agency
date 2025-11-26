package handler

import (
	"net/http"
	"strconv"
	"strings"

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

func (h *PropertyHandler) Register(mux *http.ServeMux, prefix string) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	mux.HandleFunc(prefix, h.handleProperties)
}

func (h *PropertyHandler) handleProperties(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
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
	case http.MethodGet:
		path := strings.TrimPrefix(r.URL.Path, "/properties/")
		if path == "" || path == "/" {
			writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "list not implemented"})
			return
		}
		id, err := strconv.Atoi(strings.Trim(path, "/"))
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
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
