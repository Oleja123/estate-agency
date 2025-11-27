package propertytypehandler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	auth "github.com/Oleja123/estate-agency/internal/handler/auth"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	typesvc "github.com/Oleja123/estate-agency/internal/application/property_type"
	dto "github.com/Oleja123/estate-agency/internal/application/property_type/dto"
	handlerutils "github.com/Oleja123/estate-agency/internal/handler/utils"
)

type PropertyTypeHandler struct {
	svc typesvc.Service
	lg  *slog.Logger
}

// Documentation-only DTOs for property type endpoints to simplify swagger parsing.
// Documentation-only DTOs for property type endpoints are defined in
// docs_dto.go to keep the handler file small and to make swagger
// generation more robust.

func NewPropertyTypeHandler(s typesvc.Service, l *slog.Logger) *PropertyTypeHandler {
	return &PropertyTypeHandler{svc: s, lg: l}
}

func (h *PropertyTypeHandler) Register(r chi.Router, prefix string, authMw func(next http.Handler) http.Handler) {
	if prefix == "" {
		prefix = "/property_types"
	}
	// All property_type endpoints require authorization. If the caller did not
	// provide an auth middleware we insert a denying middleware so requests
	// will be rejected rather than accidentally exposed.
	if authMw == nil {
		authMw = func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerutils.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing authorization"})
			})
		}
	}

	r.Route(prefix, func(r chi.Router) {
		// apply auth middleware for all routes under this prefix
		r.Use(authMw)

		// reads available to any authenticated user
		r.Get("/", h.handleList)
		r.Get("/{id}", h.handleGet)

		// only admins may create/update/delete property types
		r.With(auth.RequireAdminMiddleware()).Post("/", h.handleCreate)
		r.With(auth.RequireAdminMiddleware()).Put("/{id}", h.handleUpdate)
		r.With(auth.RequireAdminMiddleware()).Delete("/{id}", h.handleDelete)
	})
}

// @Security BearerAuth
// @Summary Create property type
// @Description Create a new property type
// @Tags property_types
// @Accept json
// @Produce json
// @Param body body CreatePropertyTypeDoc true "Property type body"
// @Success 201 {object} PropertyTypeDTODoc
// @Failure 400 {object} map[string]string
// @Router /property_types [post]
func (h *PropertyTypeHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req dto.CreatePropertyTypeRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	t, err := h.svc.Create(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusCreated, t)
}

// @Security BearerAuth
// @Summary Get property type
// @Description Get property type by ID
// @Tags property_types
// @Produce json
// @Param id path int true "Property Type ID"
// @Success 200 {object} PropertyTypeDTODoc
// @Failure 400 {object} map[string]string
// @Router /property_types/{id} [get]
func (h *PropertyTypeHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	t, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, t)
}

// @Security BearerAuth
// @Summary List property types
// @Description List property types with pagination
// @Tags property_types
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {object} ListPropertyTypesResponseDoc
// @Failure 400 {object} map[string]string
// @Router /property_types [get]
func (h *PropertyTypeHandler) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 20
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 500 {
		limit = 500
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	req := dto.ListPropertyTypesRequest{Limit: limit, Offset: offset}
	res, err := h.svc.List(r.Context(), req)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, res)
}

// @Security BearerAuth
// @Summary Update property type
// @Description Update property type by ID
// @Tags property_types
// @Accept json
// @Produce json
// @Param id path int true "Property Type ID"
// @Param body body UpdatePropertyTypeDoc true "Property type body"
// @Success 204
// @Failure 400 {object} map[string]string
// @Router /property_types/{id} [put]
func (h *PropertyTypeHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req dto.UpdatePropertyTypeRequest
	if err := handlerutils.DecodeJSON(r, &req); err != nil {
		code, body := handlerutils.MapAppError(apperrors.NewErrInvalidInput("body", nil, "invalid json"))
		handlerutils.WriteJSON(w, code, body)
		return
	}
	req.ID = id
	if err := h.svc.Update(r.Context(), req); err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusNoContent, nil)
}

// @Security BearerAuth
// @Summary Delete property type
// @Description Delete property type by ID
// @Tags property_types
// @Produce json
// @Param id path int true "Property Type ID"
// @Success 200 {object} map[string]int
// @Failure 400 {object} map[string]string
// @Router /property_types/{id} [delete]
func (h *PropertyTypeHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		handlerutils.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	deletedID, err := h.svc.Delete(r.Context(), id)
	if err != nil {
		code, body := handlerutils.MapAppError(err)
		handlerutils.WriteJSON(w, code, body)
		return
	}
	handlerutils.WriteJSON(w, http.StatusOK, map[string]int{"deleted_id": deletedID})
}
