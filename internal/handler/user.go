package handler

import (
	"errors"
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
	mux.HandleFunc(prefix, h.handleUsers) // list, specific by id, or profile update
}

// handleProfile handles requests to update a user's profile.
// Expected URL: PUT /users/{id}/profile
func (h *UserHandler) handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	// extract id from path: trim prefix '/users/' and suffix '/profile'
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[1] != "profile" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid path"})
		return
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req dto.UpdateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		writeJSON(w, code, body)
		return
	}
	req.UserID = id
	if err := h.svc.UpdateProfile(r.Context(), req); err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
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
		// For login specifically, map invalid credentials to 401 Unauthorized
		var inv apperrors.ErrInvalidInput
		if errors.As(err, &inv) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *UserHandler) handleUsers(w http.ResponseWriter, r *http.Request) {
	// support GET /users -> list, GET /users/{id} -> get, PUT /users/{id}/profile
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	switch r.Method {
	case http.MethodPut:
		// support PUT /users/{id}/profile
		if strings.HasSuffix(path, "/profile") {
			h.handleProfile(w, r)
			return
		}
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	case http.MethodGet:
		// check for id suffix: GET /users/{id}
		path := strings.TrimPrefix(r.URL.Path, "/users/")
		if path != "" && path != "/" && !strings.Contains(path, "list") {
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
			return
		}

		req, _ := parseListUsersRequest(r)
		res, err := h.svc.ListUsers(r.Context(), req)
		if err != nil {
			code, body := mapAppError(err)
			writeJSON(w, code, body)
			return
		}
		writeJSON(w, http.StatusOK, res)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}
