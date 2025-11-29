package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	tokensvc "github.com/Oleja123/estate-agency/internal/application/token"
	usersvc "github.com/Oleja123/estate-agency/internal/application/user"
	handlerutils "github.com/Oleja123/estate-agency/internal/handler/utils"
)

type TokenHandler struct {
	tokenSvc tokensvc.Service
	userSvc  usersvc.Service
}

type RefreshRequestDoc struct {
	RefreshToken string `json:"refresh_token"`
}

func NewTokenHandler(t tokensvc.Service, u usersvc.Service) *TokenHandler {
	return &TokenHandler{tokenSvc: t, userSvc: u}
}

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
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	// Delegate to application user service which performs validation and
	// returns a full LoginResponse (user, access_token, refresh_token, expires_at).
	resp, err := h.userSvc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid refresh token"})
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, resp)
}
