package auth

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
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
		r.Post("/logout", h.handleLogout)
	})
}

func (h *TokenHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный JSON"})
		return
	}

	resp, err := h.userSvc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		var forb apperrors.ErrForbidden
		if errors.As(err, &forb) {
			handlerutils.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "аккаунт деактивирован"})
			return
		}
		handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "недействительный refresh-токен"})
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, resp)
}

func (h *TokenHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный JSON"})
		return
	}

	err := h.userSvc.Logout(r.Context(), req.RefreshToken)
	if err != nil {
		var forb apperrors.ErrForbidden
		if errors.As(err, &forb) {
			handlerutils.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "аккаунт деактивирован"})
			return
		}
		handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "недействительный refresh-токен"})
		return
	}

	handlerutils.WriteJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}
