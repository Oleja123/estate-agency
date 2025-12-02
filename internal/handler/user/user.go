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
)

type UserHandler struct {
	svc    usersvc.Service
	lg     *slog.Logger
	favSvc favoritesvc.Service
}

func NewUserHandler(s usersvc.Service, l *slog.Logger, fav favoritesvc.Service) *UserHandler {
	return &UserHandler{svc: s, lg: l, favSvc: fav}
}

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
					r.With(auth.RequireOwnerMiddleware()).Get("/{id}/favorites", h.handleGetFavorites)
				}

				r.With(auth.RequireOwnerMiddleware()).Patch("/{id}/profile", h.handleProfile)
				r.With(auth.RequireOwnerOrAdminMiddleware()).Delete("/{id}", h.handleDeleteUser)
				r.With(auth.RequireOwnerOrAdminMiddleware()).Patch("/{id}/password", h.handleChangePassword)
				r.With(auth.RequireOwnerOrAdminMiddleware()).Patch("/{id}/active", h.handleSetActive)
				r.With(auth.RequireAdminMiddleware()).Patch("/{id}/role", h.handleChangeRole)
			})
		} else {
			r.Get("/", h.handleListUsers)
			r.Get("/{id}", h.handleGetUser)
			r.Patch("/{id}/profile", h.handleProfile)
			r.Delete("/{id}", h.handleDeleteUser)
			r.Patch("/{id}/password", h.handleChangePassword)
			r.Patch("/{id}/active", h.handleSetActive)
		}
	})
}

func (h *UserHandler) handleGetFavorites(w http.ResponseWriter, r *http.Request) {
	if h.favSvc == nil {
		handlerutils.WriteJSON(w, http.StatusNotImplemented, map[string]string{"error": "избранное не включено"})
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
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

func (h *UserHandler) handleProfile(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}

	var req dto.UpdateProfileRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "некорректный JSON"))
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

func (h *UserHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handlerutils.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "метод не разрешён"})
		return
	}
	var req dto.RegisterRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "некорректный JSON"))
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

func (h *UserHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "некорректный JSON"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	res, err := h.svc.Login(r.Context(), req)
	if err != nil {
		var inv apperrors.ErrInvalidInput
		if errors.As(err, &inv) {
			handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "неверные учётные данные"})
			return
		}
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, res)
}

func (h *UserHandler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
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

func (h *UserHandler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}

	if uid, ok := auth.UserIDFromContext(r.Context()); ok && uid == id {
		handlerutils.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "нельзя удалить самого себя"})
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

func (h *UserHandler) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}
	var req dto.ChangePasswordRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "некорректный JSON"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	if role, ok := auth.RoleFromContext(r.Context()); ok && role == "admin" {
		if req.NewPassword == "" {
			code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("new_password", nil, "не может быть пустым"))
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

func (h *UserHandler) handleSetActive(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}
	// запретить пользователю деактивировать самого себя
	if uid, ok := auth.UserIDFromContext(r.Context()); ok && uid == id {
		// получим текущее состояние пользователя — если он активен, то смена статуса приведёт к деактивации
		u, err := h.svc.GetUserByID(r.Context(), id)
		if err != nil {
			code, body := handlerutils.MapAppError(err)
			handlerutils.WriteJSON(w, code, body)
			return
		}
		if u.IsActive {
			handlerutils.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "нельзя деактивировать самого себя"})
			return
		}
		// если пользователь уже неактивен — позволим toggling (это активирует его)
	}

	if err := h.svc.ToggleActiveAccount(r.Context(), id); err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}

	// после успешного переключения статуса вернём обновлённого пользователя,
	// чтобы клиент мог обновить список без дополнительного запроса
	u, err := h.svc.GetUserByID(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, u)
}

func (h *UserHandler) handleChangeRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "некорректный id"})
		return
	}

	if uid, ok := auth.UserIDFromContext(r.Context()); ok && uid == id {
		handlerutils.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "нельзя изменить свою роль"})
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "некорректный JSON"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	if req.Role == "" {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("role", req.Role, "не может быть пустым"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	if err := h.svc.SetUserRole(r.Context(), id, req.Role); err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusNoContent, nil)
}
