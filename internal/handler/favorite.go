package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	favoritesvc "github.com/Oleja123/estate-agency/internal/application/favorite"
	dto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
)

type FavoriteHandler struct {
	svc favoritesvc.Service
}

func NewFavoriteHandler(s favoritesvc.Service) *FavoriteHandler {
	return &FavoriteHandler{svc: s}
}

func (h *FavoriteHandler) Register(r chi.Router, prefix string) {
	if prefix == "" {
		prefix = "/favorites"
	}
	r.Route(prefix, func(r chi.Router) {
		r.Post("/", h.handleFavorites)
		r.Get("/", h.handleFavorites)
		r.Delete("/", h.handleFavorites)
	})
}

func (h *FavoriteHandler) handleFavorites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req dto.CreateFavoriteRequest
		if err := decodeJSON(r, &req); err != nil {
			code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
			writeJSON(w, code, body)
			return
		}
		f, err := h.svc.Create(r.Context(), req)
		if err != nil {
			code, body := mapAppError(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusCreated, f)
	case http.MethodGet:
		q := r.URL.Query()
		if q.Get("user_id") != "" && q.Get("property_id") != "" {
			uid, _ := strconv.Atoi(q.Get("user_id"))
			pid, _ := strconv.Atoi(q.Get("property_id"))
			res, err := h.svc.GetByUserAndProperty(r.Context(), dto.CreateFavoriteRequest{UserID: uid, PropertyID: pid})
			if err != nil {
				code, body := mapAppError(err)
				writeJSON(w, code, body)
				return
			}
			writeJSON(w, http.StatusOK, res)
			return
		}
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "listing not implemented"})
	case http.MethodDelete:
		q := r.URL.Query()
		uid, _ := strconv.Atoi(q.Get("user_id"))
		pid, _ := strconv.Atoi(q.Get("property_id"))
		if uid == 0 || pid == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing ids"})
			return
		}
		_, err := h.svc.Delete(r.Context(), dto.CreateFavoriteRequest{UserID: uid, PropertyID: pid})
		if err != nil {
			code, body := mapAppError(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusNoContent, nil)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
