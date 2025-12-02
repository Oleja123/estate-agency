package apperrors

import "fmt"

type ErrInvalidInput struct {
	Field  string
	Value  interface{}
	Reason string
}

func NewErrInvalidInput(field string, value interface{}, reason string) error {
	return ErrInvalidInput{Field: field, Value: value, Reason: reason}
}

func (e ErrInvalidInput) Error() string {
	return fmt.Sprintf("некорректное поле '%s': %s (значение: %v)", e.Field, e.Reason, e.Value)
}

type ErrAlreadyExists struct {
	Entity string
	Field  string
	Value  interface{}
}

func NewErrAlreadyExists(entity, field string, value interface{}) error {
	return ErrAlreadyExists{Entity: entity, Field: field, Value: value}
}

func (e ErrAlreadyExists) Error() string {
	return fmt.Sprintf("%s с %s '%v' уже существует", e.Entity, e.Field, e.Value)
}

type ErrNotFound struct {
	Entity string
	Id     interface{}
}

func NewErrNotFound(entity string, id interface{}) error {
	return ErrNotFound{Entity: entity, Id: id}
}

func (e ErrNotFound) Error() string {
	if e.Id != nil {
		return fmt.Sprintf("%s с %v не найдено", e.Entity, e.Id)
	}
	return fmt.Sprintf("%s не найдено", e.Entity)
}

type ErrInternal struct {
	Message string
}

func NewErrInternal(message string) error {
	return ErrInternal{Message: message}
}

func (e ErrInternal) Error() string {
	return fmt.Sprintf("internal error: %s", e.Message)
}

type ErrTimeout struct {
	Message string
}

func NewErrTimeout(message string) error {
	return ErrTimeout{Message: message}
}

func (e ErrTimeout) Error() string {
	return fmt.Sprintf("timeout: %s", e.Message)
}

type ErrGeocoding struct {
	Message string
}

func NewErrGeocoding(message string) error {
	return ErrGeocoding{Message: message}
}

func (e ErrGeocoding) Error() string {
	return fmt.Sprintf("ошибка геокодирования: %s", e.Message)
}

type ErrForbidden struct {
	Message string
}

func NewErrForbidden(message string) error {
	return ErrForbidden{Message: message}
}

func (e ErrForbidden) Error() string {
	return fmt.Sprintf("forbidden: %s", e.Message)
}
