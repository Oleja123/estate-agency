package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	tokensvc "github.com/Oleja123/estate-agency/internal/application/token"
	handlerutils "github.com/Oleja123/estate-agency/internal/handler/utils"
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
	// @Summary Refresh token
	// @Description Validate and refresh a refresh token
	// @Tags tokens
	// @Accept json
	// @Produce json
	// @Param body body object true "Refresh token"
	// @Success 200 {object} map[string]int
	// @Failure 400 {object} map[string]string
	// @Failure 401 {object} map[string]string
	// @Router /tokens/refresh [post]
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	res, err := h.svc.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, map[string]int{"user_id": res})
}
