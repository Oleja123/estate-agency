package userhandler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	favoritesvc "github.com/Oleja123/estate-agency/internal/application/favorite"
	favdto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	usersvc "github.com/Oleja123/estate-agency/internal/application/user"
	dto "github.com/Oleja123/estate-agency/internal/application/user/dto"
	"github.com/Oleja123/estate-agency/internal/handler/auth"
	handlerutils "github.com/Oleja123/estate-agency/internal/handler/utils"
	optional "github.com/denpa16/optional-go-type"
)

type UserHandler struct {
	svc    usersvc.Service
	lg     *slog.Logger
	favSvc favoritesvc.Service
}

// Documentation-only DTOs used for swagger generation. Kept local to avoid
// complex cross-package parsing issues.
// Documentation-only DTOs are defined in docs_dto.go to keep handler files
// small and to avoid redeclaration when generating swagger.
// NewUserHandler creates a new UserHandler.
// The favorites service may be provided as an optional third argument. This
// keeps existing two-argument calls working in tests while allowing main to
// pass the favorites service via constructor.
func NewUserHandler(s usersvc.Service, l *slog.Logger, fav favoritesvc.Service) *UserHandler {
	return &UserHandler{svc: s, lg: l, favSvc: fav}
}

// Register attaches user routes under the given prefix. Example prefix: "/users"
func (h *UserHandler) Register(r chi.Router, prefix string, authMw func(next http.Handler) http.Handler) {
	if prefix == "" {
		prefix = "/users"
	}
	r.Route(prefix, func(r chi.Router) {
		r.Post("/register", h.handleRegister)
		r.Post("/login", h.handleLogin)

		if authMw != nil {
			r.Group(func(r chi.Router) {
				r.Use(authMw)
				r.With(auth.RequireAdminMiddleware()).Get("/", h.handleListUsers)
				r.With(auth.RequireOwnerOrAdminMiddleware()).Get("/{id}", h.handleGetUser)

				if h.favSvc != nil {
					// Allow only the owner (not admin) to view a user's favorites
					r.With(auth.RequireOwnerMiddleware()).Get("/{id}/favorites", h.handleGetFavorites)
				}

				r.With(auth.RequireOwnerMiddleware()).Put("/{id}/profile", h.handleProfile)
				r.With(auth.RequireOwnerOrAdminMiddleware()).Delete("/{id}", h.handleDeleteUser)
				r.With(auth.RequireOwnerOrAdminMiddleware()).Post("/{id}/password", h.handleChangePassword)
				// single endpoint to set active/inactive state
				r.With(auth.RequireOwnerOrAdminMiddleware()).Post("/{id}/active", h.handleSetActive)
				r.With(auth.RequireAdminMiddleware()).Post("/{id}/role", h.handleChangeRole)
			})
		} else {
			r.Get("/", h.handleListUsers)
			r.Get("/{id}", h.handleGetUser)
			r.Put("/{id}/profile", h.handleProfile)
			r.Delete("/{id}", h.handleDeleteUser)
			r.Post("/{id}/password", h.handleChangePassword)
			r.Post("/{id}/active", h.handleSetActive)
		}
	})
}

// SetFavoriteService provides a way to attach the favorites service after
// construction. It's kept for compatibility, but constructor injection is
// preferred.
// constructor-only injection enforced; SetFavoriteService removed

// @Security BearerAuth
// @Summary Get favorites for a user
// @Description List favorites for the specified user. Only the owner may view their favorites.
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} object
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/{id}/favorites [get]
// handleGetFavorites handles GET /users/{id}/favorites
func (h *UserHandler) handleGetFavorites(w http.ResponseWriter, r *http.Request) {
	if h.favSvc == nil {
		handlerutils.WriteJSON(w, http.StatusNotImplemented, map[string]string{"error": "favorites not enabled"})
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	req := favdto.ListFavoritesRequest{Limit: 500, Offset: 0}
	req.Filter.UserID = id
	res, err := h.favSvc.List(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, res)
}

// handleProfile handles requests to update a user's profile.
// Expected URL: PUT /users/{id}/profile
// @Security BearerAuth
// @Summary Update user profile
// @Description Update profile for a user (owner or admin)
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param body body UpdateProfileRequestDoc true "Profile"
// @Success 204
// @Failure 400 {object} map[string]string
// @Router /users/{id}/profile [put]
func (h *UserHandler) handleProfile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req dto.UpdateProfileRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	req.UserID = id
	if err := h.svc.UpdateProfile(r.Context(), req); err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusNoContent, nil)
}

// The rest of the file implements the usual user endpoints (register, login,
// list, get, profile update, delete, password change, activate/deactivate, role change).
// They mirror the project's existing behavior and use handlerutils and the
// user service to perform actions and map errors.

// @Summary Register user
// @Description Register a new user
// @Tags users
// @Accept json
// @Produce json
// @Param body body RegisterRequestDoc true "Register"
// @Success 201 {object} PublicUserDoc
// @Failure 400 {object} map[string]string
// @Router /users/register [post]
func (h *UserHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handlerutils.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var req dto.RegisterRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	u, err := h.svc.Register(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusCreated, u)
}

// @Summary Login
// @Description Authenticate user and return tokens
// @Tags users
// @Accept json
// @Produce json
// @Param body body LoginRequestDoc true "Login credentials"
// @Success 200 {object} LoginResponseDoc
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /users/login [post]
func (h *UserHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		// For login specifically, map invalid credentials to 401 Unauthorized
		var inv apperrors.ErrInvalidInput
		if errors.As(err, &inv) {
			handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, res)
}

// @Security BearerAuth
// @Summary Get user
// @Description Get user by ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} PublicUserDoc
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /users/{id} [get]
func (h *UserHandler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	u, err := h.svc.GetUserByID(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, u)
}

// @Security BearerAuth
// @Security BearerAuth
// @Summary List users
// @Description List users with pagination and optional filters
// @Tags users
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Param email query string false "Filter by exact email"
// @Param role query string false "Filter by role (admin/user)"
// @Param search query string false "Search in name and email"
// @Param is_active query boolean false "Filter by active state (true/false)"
// @Param ids query string false "Comma-separated list of user IDs to include"
// @Success 200 {object} ListUsersResponseDoc
// @Router /users [get]
func (h *UserHandler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	req, _ := parseListUsersRequest(r)
	res, err := h.svc.ListUsers(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, res)
}

// @Security BearerAuth
// @Summary Delete user
// @Description Delete user by ID
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]int
// @Failure 400 {object} map[string]string
// @Router /users/{id} [delete]
func (h *UserHandler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	deletedID, err := h.svc.DeleteUser(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, map[string]int{"deleted_id": deletedID})
}

// @Security BearerAuth
// @Summary Change user password
// @Description Change password for a user (owner or admin)
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param body body ChangePasswordRequestDoc true "Password request"
// @Success 204
// @Failure 400 {object} map[string]string
// @Router /users/{id}/password [post]
func (h *UserHandler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req dto.ChangePasswordRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	if role, ok := auth.RoleFromContext(r.Context()); ok && role == "admin" {
		if req.NewPassword == "" {
			code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("new_password", nil, "must not be empty"))
			handlerutils.WriteJSON(w, code, body)
			return
		}
		if err := h.svc.ChangePasswordAdmin(r.Context(), id, req.NewPassword); err != nil {
			code, body := handlerutils.MapAppError(err)
			handlerutils.WriteJSON(w, code, body)
			return
		}
		handlerutils.WriteJSON(w, http.StatusNoContent, nil)
		return
	}
	if err := h.svc.ChangePassword(r.Context(), id, req); err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusNoContent, nil)
}

// handleSetActive handles POST /users/{id}/active with body {"active": true|false}
// This combines previous activate/deactivate endpoints into one.
// @Security BearerAuth
// @Summary Toggle active state
// @Description Toggle user's active state (owner or admin)
// @Tags users
// @Param id path int true "User ID"
// @Success 204
// @Failure 400 {object} map[string]string
// @Router /users/{id}/active [post]
func (h *UserHandler) handleSetActive(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	// Delegate toggle behaviour to application service to keep handler thin.
	if err := h.svc.ToggleActiveAccount(r.Context(), id); err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}

	handlerutils.WriteJSON(w, http.StatusNoContent, nil)
}

// @Security BearerAuth
// @Summary Change user role
// @Description Change role for a user (admin only)
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param body body RoleRequestDoc true "Role body"
// @Success 204
// @Failure 400 {object} map[string]string
// @Router /users/{id}/role [post]
func (h *UserHandler) handleChangeRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	if req.Role == "" {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("role", req.Role, "must not be empty"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	roleVal := req.Role
	upr := dto.UpdateProfileRequest{UserID: id, Role: optional.OptionalString{Defined: true, Valid: true, Value: &roleVal}}
	if err := h.svc.UpdateProfile(r.Context(), upr); err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusNoContent, nil)
}
