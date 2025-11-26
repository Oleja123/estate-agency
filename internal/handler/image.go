package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

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

func (h *ImageHandler) Register(r chi.Router, prefix string) {
	if prefix == "" {
		prefix = "/images"
	}
	r.Route(prefix, func(r chi.Router) {
		r.Post("/", h.handleCreateMany)
		r.Get("/", h.handleListByProperty)
	})
}

func (h *ImageHandler) handleCreateMany(w http.ResponseWriter, r *http.Request) {
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
}

func (h *ImageHandler) handleListByProperty(w http.ResponseWriter, r *http.Request) {
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
}
