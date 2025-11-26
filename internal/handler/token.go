package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	tokensvc "github.com/Oleja123/estate-agency/internal/application/token"
)

type TokenHandler struct {
	svc tokensvc.Service
}

func NewTokenHandler(s tokensvc.Service) *TokenHandler {
	return &TokenHandler{svc: s}
}

// Token endpoints: generate/refresh/validate. For brevity only refresh is sketched.
func (h *TokenHandler) Register(r chi.Router, prefix string, _ func(next http.Handler) http.Handler) {
	if prefix == "" {
		prefix = "/tokens"
	}
	r.Route(prefix, func(r chi.Router) {
		r.Post("/refresh", h.handleRefresh)
	})
}

func (h *TokenHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := h.svc.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"user_id": res})
}
