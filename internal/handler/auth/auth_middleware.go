package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"log/slog"

	"github.com/Oleja123/estate-agency/internal/application/token"
	handlerutils "github.com/Oleja123/estate-agency/internal/handler/utils"
	"github.com/go-chi/chi/v5"
)

type ctxKey string

const ctxKeyUserID ctxKey = "user_id"
const ctxKeyUserRole ctxKey = "user_role"

func UserIDFromContext(ctx context.Context) (int, bool) {
	v := ctx.Value(ctxKeyUserID)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int)
	return id, ok
}

func AuthMiddleware(tokSvc token.Service) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lg := pkgLogger
			auth := r.Header.Get("Authorization")
			if lg != nil {
				lg.Info("auth middleware check", "method", r.Method, "path", r.URL.Path, "auth_present", auth != "")
			}
			if auth == "" {
				if lg != nil {
					lg.Info("auth middleware: missing Authorization header", "method", r.Method, "path", r.URL.Path)
				}
				handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
				return
			}
			var tokenStr string
			lower := strings.ToLower(auth)
			if strings.HasPrefix(lower, "bearer ") {
				tokenStr = strings.TrimSpace(auth[len("Bearer "):])
			} else {
				if lg != nil {
					lg.Info("auth middleware: Authorization header missing scheme, treating value as raw token", "header_sample", auth[:min(len(auth), 32)])
				}
				tokenStr = strings.TrimSpace(auth)
			}
			if lg != nil {
				lg.Info("auth middleware: received token", "token_len", len(tokenStr))
			}
			uid, role, err := tokSvc.ValidateAccessToken(tokenStr)
			if err != nil {
				if lg != nil {
					lg.Info("auth middleware: token validation failed", "err", err)
				}
				handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
				return
			}
			if lg != nil {
				lg.Info("auth middleware: token valid", "user_id", uid, "role", role)
			}
			ctx := context.WithValue(r.Context(), ctxKeyUserID, uid)
			ctx = context.WithValue(ctx, ctxKeyUserRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RoleFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxKeyUserRole)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func ContextWithUser(ctx context.Context, userID int, role string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	ctx = context.WithValue(ctx, ctxKeyUserRole, role)
	return ctx
}

var pkgLogger *slog.Logger

func SetLogger(l *slog.Logger) {
	pkgLogger = l
}

func RequireOwnerMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lg := pkgLogger
			if lg == nil {
			}
			uid, ok := UserIDFromContext(r.Context())
			if !ok {
				if lg != nil {
					lg.Info("require-owner: missing user id in context", "path", r.URL.Path)
				}
				handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
				return
			}
			idStr := chi.URLParam(r, "id")
			if idStr == "" {
				if lg != nil {
					lg.Info("require-owner: missing id param", "path", r.URL.Path)
				}
				handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id"})
				return
			}
			id, err := strconv.Atoi(idStr)
			if err != nil {
				if lg != nil {
					lg.Info("require-owner: invalid id param", "id", idStr, "err", err)
				}
				handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
				return
			}
			if id != uid {
				if lg != nil {
					lg.Info("require-owner: forbidden - user mismatch", "path", r.URL.Path, "user_id", uid, "target_id", id)
				}
				handlerutils.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireOwnerOrAdminMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lg := pkgLogger
			uid, ok := UserIDFromContext(r.Context())
			if !ok {
				if lg != nil {
					lg.Info("require-owner-or-admin: missing user id in context", "path", r.URL.Path)
				}
				handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
				return
			}
			role, _ := RoleFromContext(r.Context())
			if role == "admin" {
				if lg != nil {
					lg.Info("require-owner-or-admin: admin allowed", "user_id", uid, "path", r.URL.Path)
				}
				next.ServeHTTP(w, r)
				return
			}
			idStr := chi.URLParam(r, "id")
			if idStr == "" {
				if lg != nil {
					lg.Info("require-owner-or-admin: missing id param", "path", r.URL.Path)
				}
				handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing id"})
				return
			}
			id, err := strconv.Atoi(idStr)
			if err != nil {
				if lg != nil {
					lg.Info("require-owner-or-admin: invalid id param", "id", idStr, "err", err)
				}
				handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
				return
			}
			if id != uid {
				if lg != nil {
					lg.Info("require-owner-or-admin: forbidden - user mismatch", "path", r.URL.Path, "user_id", uid, "target_id", id)
				}
				handlerutils.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdminMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lg := pkgLogger
			uid, ok := UserIDFromContext(r.Context())
			if !ok {
				if lg != nil {
					lg.Info("require-admin: missing user id in context", "path", r.URL.Path)
				}
				handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
				return
			}
			role, _ := RoleFromContext(r.Context())
			if role != "admin" {
				if lg != nil {
					lg.Info("require-admin: forbidden - not admin", "user_id", uid, "role", role)
				}
				handlerutils.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
