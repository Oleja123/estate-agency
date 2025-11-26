package handler

import (
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	imagesvc "github.com/Oleja123/estate-agency/internal/application/image"
	dto "github.com/Oleja123/estate-agency/internal/application/image/dto"
)

type ImageHandler struct {
	svc imagesvc.Service
}

func NewImageHandler(s imagesvc.Service) *ImageHandler {
	return &ImageHandler{svc: s}
}

func (h *ImageHandler) Register(mux *http.ServeMux, prefix string) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	mux.HandleFunc(prefix, h.handleImages)
}

func (h *ImageHandler) handleImages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req dto.CreateImagesRequest
		if err := decodeJSON(r, &req); err != nil {
			code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
			writeJSON(w, code, body)
			return
		}
		imgs, err := h.svc.CreateMany(r.Context(), req)
		if err != nil {
			code, body := mapAppError(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusCreated, imgs)
	case http.MethodGet:
		// GET /images?property_id=123
		q := r.URL.Query()
		pid, _ := strconv.Atoi(q.Get("property_id"))
		if pid == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing property_id"})
			return
		}
		imgs, err := h.svc.ListByProperty(r.Context(), pid)
		if err != nil {
			code, body := mapAppError(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusOK, imgs)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
