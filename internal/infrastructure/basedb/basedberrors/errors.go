package basedberrors

import (
	"fmt"
)

type ErrNotFound struct {
	Entity string
	Id     interface{}
}

func NewErrNotFound(entity string, id interface{}) error {
	return ErrNotFound{
		Entity: entity,
		Id:     id,
	}
}

func (e ErrNotFound) Error() string {
	if e.Id != nil {
		return fmt.Sprintf("%s с %v не найдено", e.Entity, e.Id)
	}
	return fmt.Sprintf("%s не найдено", e.Entity)
}

type ErrAlreadyExists struct {
	Entity string
	Field  string
	Value  interface{}
}

func NewErrAlreadyExists(entity, field string, value interface{}) error {
	return ErrAlreadyExists{
		Entity: entity,
		Field:  field,
		Value:  value,
	}
}

func (e ErrAlreadyExists) Error() string {
	return fmt.Sprintf("%s с %s '%v' уже существует", e.Entity, e.Field, e.Value)
}

type ErrForeignKeyViolation struct {
	Table      string
	Constraint string
	Key        interface{}
}

func NewErrForeignKeyViolation(table, constraint string, key interface{}) error {
	return ErrForeignKeyViolation{
		Table:      table,
		Constraint: constraint,
		Key:        key,
	}
}

func (e ErrForeignKeyViolation) Error() string {
	return fmt.Sprintf("нарушение внешнего ключа в %s (ограничение: %s, ключ: %v)",
		e.Table, e.Constraint, e.Key)
}

type ErrInvalidInput struct {
	Field  string
	Value  interface{}
	Reason string
}

func NewErrInvalidInput(field string, value interface{}, reason string) error {
	return ErrInvalidInput{
		Field:  field,
		Value:  value,
		Reason: reason,
	}
}

func (e ErrInvalidInput) Error() string {
	return fmt.Sprintf("некорректное поле '%s': %s (значение: %v)",
		e.Field, e.Reason, e.Value)
}

type ErrDatabase struct {
	Operation string
	Details   string
}

func NewErrDatabase(operation, details string) error {
	return ErrDatabase{
		Operation: operation,
		Details:   details,
	}
}

func (e ErrDatabase) Error() string {
	return fmt.Sprintf("ошибка базы данных во время операции %s: %s", e.Operation, e.Details)
}

type ErrTimeout struct {
	Operation string
	Timeout   string
}

func NewErrTimeout(operation, timeout string) error {
	return ErrTimeout{
		Operation: operation,
		Timeout:   timeout,
	}
}

func (e ErrTimeout) Error() string {
	return fmt.Sprintf("операция '%s' превысила время ожидания %s", e.Operation, e.Timeout)
}

type ErrConnection struct {
	Details string
}

func NewErrConnection(details string) error {
	return ErrConnection{
		Details: details,
	}
}

func (e ErrConnection) Error() string {
	return fmt.Sprintf("ошибка подключения к базе данных: %s", e.Details)
}
