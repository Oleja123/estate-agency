package handler

import (
	"net/http"
	"strconv"
	"strings"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	usersvc "github.com/Oleja123/estate-agency/internal/application/user"
	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
)

type UserHandler struct {
	svc usersvc.Service
}

func NewUserHandler(s usersvc.Service) *UserHandler {
	return &UserHandler{svc: s}
}

// Register attaches user routes under the given prefix. Example prefix: "/users"
func (h *UserHandler) Register(mux *http.ServeMux, prefix string) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	mux.HandleFunc(prefix+"register", h.handleRegister)
	mux.HandleFunc(prefix+"login", h.handleLogin)
	mux.HandleFunc(prefix, h.handleUsers) // list or specific by id
}

func (h *UserHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req dto.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		writeJSON(w, code, body)
		return
	}
	u, err := h.svc.Register(r.Context(), req)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (h *UserHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		writeJSON(w, code, body)
		return
	}
	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *UserHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	// support GET /users -> list, GET /users/{id} -> get, PUT/DELETE /users/{id}
	switch r.Method {
	case http.MethodGet:
		// check for id suffix
		path := strings.TrimPrefix(r.URL.Path, "/users/")
		if path == "" || path == "/" || strings.Contains(path, "list") {
			// TODO: list users (not implemented here)
			writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "list not implemented"})
			return
		}
		id, err := strconv.Atoi(strings.Trim(path, "/"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}
		u, err := h.svc.GetUserByID(r.Context(), id, 0)
		if err != nil {
			code, body := mapAppError(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusOK, u)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
