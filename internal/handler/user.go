package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	usersvc "github.com/Oleja123/estate-agency/internal/application/user"
	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
	optional "github.com/denpa16/optional-go-type"
)

type UserHandler struct {
	svc usersvc.Service
}

func NewUserHandler(s usersvc.Service) *UserHandler {
	return &UserHandler{svc: s}
}

// Register attaches user routes under the given prefix. Example prefix: "/users"
func (h *UserHandler) Register(r chi.Router, prefix string, authMw func(next http.Handler) http.Handler) {
	// ensure prefix starts with '/'
	if prefix == "" {
		prefix = "/users"
	}
	r.Route(prefix, func(r chi.Router) {
		r.Post("/register", h.handleRegister)
		r.Post("/login", h.handleLogin)

		// If auth middleware is provided, protect listing and get-by-id routes
		if authMw != nil {
			r.Group(func(r chi.Router) {
				r.Use(authMw)
				// list users - admin only
				r.With(RequireAdminMiddleware()).Get("/", h.handleListUsers)
				// get user - owner or admin
				r.With(RequireOwnerOrAdminMiddleware()).Get("/{id}", h.handleGetUser)

				// Only the owner of the profile may update it
				r.With(RequireOwnerMiddleware()).Put("/{id}/profile", h.handleProfile)
				// account management
				// delete/password/deactivate allowed for owner or admin
				r.With(RequireOwnerOrAdminMiddleware()).Delete("/{id}", h.handleDeleteUser)
				r.With(RequireOwnerOrAdminMiddleware()).Post("/{id}/password", h.handleChangePassword)
				r.With(RequireOwnerOrAdminMiddleware()).Post("/{id}/deactivate", h.handleDeactivate)
				// role change allowed for admin only
				r.With(RequireAdminMiddleware()).Post("/{id}/role", h.handleChangeRole)
			})
		} else {
			// no auth - keep existing public behavior
			r.Get("/", h.handleListUsers)
			r.Get("/{id}", h.handleGetUser)
			r.Put("/{id}/profile", h.handleProfile)
			r.Delete("/{id}", h.handleDeleteUser)
			r.Post("/{id}/password", h.handleChangePassword)
			r.Post("/{id}/deactivate", h.handleDeactivate)
		}
	})
}

// handleProfile handles requests to update a user's profile.
// Expected URL: PUT /users/{id}/profile
func (h *UserHandler) handleProfile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
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
	// POST /users/login
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

func (h *UserHandler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	u, err := h.svc.GetUserByID(r.Context(), id)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (h *UserHandler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	req, _ := parseListUsersRequest(r)
	res, err := h.svc.ListUsers(r.Context(), req)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// handleDeleteUser handles DELETE /users/{id}
func (h *UserHandler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	deletedID, err := h.svc.DeleteUser(r.Context(), id)
	if err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"deleted_id": deletedID})
}

// handleChangePassword handles POST /users/{id}/password
func (h *UserHandler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req dto.ChangePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		writeJSON(w, code, body)
		return
	}
	req.UserID = id
	if err := h.svc.ChangePassword(r.Context(), req); err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// handleDeactivate handles POST /users/{id}/deactivate
func (h *UserHandler) handleDeactivate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := h.svc.DeactivateAccount(r.Context(), id); err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// handleChangeRole handles POST /users/{id}/role
// Expected body: { "role": "admin" }
func (h *UserHandler) handleChangeRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &req); err != nil {
		code, body := mapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		writeJSON(w, code, body)
		return
	}
	// build UpdateProfileRequest with Role set
	if req.Role == "" {
		code, body := mapAppError(apperrors.NewErrInvalidInput("role", req.Role, "must not be empty"))
		writeJSON(w, code, body)
		return
	}
	roleVal := req.Role
	upr := dto.UpdateProfileRequest{UserID: id, Role: optional.OptionalString{Defined: true, Valid: true, Value: &roleVal}}

	if err := h.svc.UpdateProfile(r.Context(), upr); err != nil {
		code, body := mapAppError(err)
		writeJSON(w, code, body)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
