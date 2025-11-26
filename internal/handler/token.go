package handler

import (
	"net/http"

	tokensvc "github.com/Oleja123/estate-agency/internal/application/token"
)

type TokenHandler struct {
	svc tokensvc.Service
}

func NewTokenHandler(s tokensvc.Service) *TokenHandler {
	return &TokenHandler{svc: s}
}

// Token endpoints: generate/refresh/validate. For brevity only refresh is sketched.
func (h *TokenHandler) Register(mux *http.ServeMux, prefix string) {
	if prefix == "" {
		prefix = "/tokens/"
	}
	if prefix[len(prefix)-1] != '/' {
		prefix += "/"
	}
	mux.HandleFunc(prefix+"refresh", h.handleRefresh)
}

func (h *TokenHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// simple: expect JSON {"refresh_token": "..."}
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
