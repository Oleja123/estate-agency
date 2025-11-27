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

// context key for user id
type ctxKey string

const ctxKeyUserID ctxKey = "user_id"
const ctxKeyUserRole ctxKey = "user_role"

// UserIDFromContext returns user id stored by the auth middleware.
func UserIDFromContext(ctx context.Context) (int, bool) {
	v := ctx.Value(ctxKeyUserID)
	if v == nil {
		return 0, false
	}
	id, ok := v.(int)
	return id, ok
}

// AuthMiddleware returns a middleware that checks Authorization header for
// a Bearer token and validates it using the provided token service. On
// success the middleware stores user id into request context under
// UserIDFromContext key. On failure it responds with 401.
// The middleware is intended to be applied per-route (chi groups). It no
// longer supports a global skip list — public endpoints should be mounted
// without this middleware.
// AuthMiddleware accepts a token service and a logger. It returns a chi-style
// middleware that validates Authorization header and injects user id into
// context. Use per-route (chi group) application for protected routes.
func AuthMiddleware(tokSvc token.Service) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// use package logger if available
			lg := pkgLogger
			// debug logging to help troubleshoot auth issues
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
			// Accept both "Bearer <token>" and raw token (some clients put only the token
			// into the Authorization header). Prefer the Bearer scheme when present.
			lower := strings.ToLower(auth)
			if strings.HasPrefix(lower, "bearer ") {
				tokenStr = strings.TrimSpace(auth[len("Bearer "):])
			} else {
				// No scheme provided — accept the whole header as token but log a hint.
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

// RoleFromContext returns user role stored by the auth middleware.
func RoleFromContext(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxKeyUserRole)
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// ContextWithUser returns a new context with user id and role set. This is
// intended as a helper for other handler packages and tests to create a
// request context that mimics the auth middleware.
func ContextWithUser(ctx context.Context, userID int, role string) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUserID, userID)
	ctx = context.WithValue(ctx, ctxKeyUserRole, role)
	return ctx
}

// package-level logger to be set by the application bootstrap so middleware
// can log consistently with services.
var pkgLogger *slog.Logger

// SetLogger sets package logger used by handler middlewares.
func SetLogger(l *slog.Logger) {
	pkgLogger = l
}

// RequireOwnerMiddleware returns a middleware that allows access only if the
// authenticated user id (from context) matches the `{id}` URL parameter.
// It should be applied after the auth middleware that injects the user id.
func RequireOwnerMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ensure logger is available
			lg := pkgLogger
			if lg == nil {
				// fallback to nothing
			}
			uid, ok := UserIDFromContext(r.Context())
			if !ok {
				if lg != nil {
					lg.Info("require-owner: missing user id in context", "path", r.URL.Path)
				}
				handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
				return
			}
			// extract id from URL param
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

// RequireOwnerOrAdminMiddleware allows access when the authenticated user is
// the owner (id matches URL {id}) or has role "admin".
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
			// allow admin
			if role == "admin" {
				if lg != nil {
					lg.Info("require-owner-or-admin: admin allowed", "user_id", uid, "path", r.URL.Path)
				}
				next.ServeHTTP(w, r)
				return
			}
			// otherwise check owner
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

// RequireAdminMiddleware allows access only to users with role "admin".
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
