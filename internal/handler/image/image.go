package imagehandler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	imagesvc "github.com/Oleja123/estate-agency/internal/application/image"
	dto "github.com/Oleja123/estate-agency/internal/application/image/dto"
	handlerutils "github.com/Oleja123/estate-agency/internal/handler/utils"
)

type ImageHandler struct {
	svc imagesvc.Service
	lg  *slog.Logger
}

func NewImageHandler(s imagesvc.Service, l *slog.Logger) *ImageHandler {
	return &ImageHandler{svc: s, lg: l}
}

func (h *ImageHandler) Register(r chi.Router, prefix string, authMw func(next http.Handler) http.Handler) {
	if prefix == "" {
		prefix = "/images"
	}
	r.Route(prefix, func(r chi.Router) {
		// public reads
		r.Get("/", h.handleListByProperty)
		r.Get("/{id}", h.handleGetByID)

		if authMw != nil {
			r.Group(func(r chi.Router) {
				r.Use(authMw)
				r.Post("/", h.handleCreateMany)
				r.Post("/create", h.handleCreate)
				r.Delete("/{id}", h.handleDelete)
			})
		} else {
			r.Post("/", h.handleCreateMany)
			r.Post("/create", h.handleCreate)
			r.Delete("/{id}", h.handleDelete)
		}
	})
}

func (h *ImageHandler) handleCreateMany(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateImagesRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	imgs, err := h.svc.CreateMany(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusCreated, imgs)
}

func (h *ImageHandler) handleListByProperty(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pid, _ := strconv.Atoi(q.Get("property_id"))
	if pid == 0 {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing property_id"})
		return
	}
	imgs, err := h.svc.ListByProperty(r.Context(), pid)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, imgs)
}

func (h *ImageHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateImageRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	img, err := h.svc.Create(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusCreated, img)
}

func (h *ImageHandler) handleGetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	img, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, img)
}

func (h *ImageHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
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
