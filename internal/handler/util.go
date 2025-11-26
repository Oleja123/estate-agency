package handler

import (
	"encoding/json"
	"net/http"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func mapAppError(err error) (int, interface{}) {
	if err == nil {
		return http.StatusOK, nil
	}
	switch e := err.(type) {
	case apperrors.ErrInvalidInput:
		return http.StatusBadRequest, map[string]string{"error": e.Error()}
	case apperrors.ErrAlreadyExists:
		return http.StatusConflict, map[string]string{"error": e.Error()}
	case apperrors.ErrNotFound:
		return http.StatusNotFound, map[string]string{"error": e.Error()}
	case apperrors.ErrInternal:
		return http.StatusInternalServerError, map[string]string{"error": "internal error"}
	default:
		return http.StatusInternalServerError, map[string]string{"error": "internal error"}
	}
}

// Exported wrappers so other packages (handler subpackages/tests) can use the
// same helpers without duplicating code.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) { writeJSON(w, status, v) }
func DecodeJSON(r *http.Request, v interface{}) error            { return decodeJSON(r, v) }
func MapAppError(err error) (int, interface{})                   { return mapAppError(err) }
