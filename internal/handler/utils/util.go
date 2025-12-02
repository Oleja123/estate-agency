package handlerutils

import (
	"encoding/json"
	"fmt"
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

		return http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("некорректное поле '%s'", e.Field)}
	case apperrors.ErrAlreadyExists:
		return http.StatusConflict, map[string]string{"error": e.Error()}
	case apperrors.ErrNotFound:
		return http.StatusNotFound, map[string]string{"error": e.Error()}
	case apperrors.ErrForbidden:
		return http.StatusForbidden, map[string]string{"error": "доступ запрещён"}
	case apperrors.ErrInternal:
		return http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка"}
	default:
		return http.StatusInternalServerError, map[string]string{"error": "внутренняя ошибка"}
	}
}

func WriteJSON(w http.ResponseWriter, status int, v interface{}) { writeJSON(w, status, v) }
func DecodeJSON(r *http.Request, v interface{}) error            { return decodeJSON(r, v) }
func MapAppError(err error) (int, interface{})                   { return mapAppError(err) }
