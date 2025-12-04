package filestore

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

type ErrUnsupportedFormat struct {
	Filename string
	Detected string
}

func NewErrUnsupportedFormat(filename, detected string) error {
	return ErrUnsupportedFormat{Filename: filename, Detected: detected}
}

func (e ErrUnsupportedFormat) Error() string {
	return fmt.Sprintf("не поддерживаемый формат изображения для файла '%s': %s", e.Filename, e.Detected)
}

type ErrStorage struct {
	Operation string
	Details   string
}

func NewErrStorage(op, details string) error {
	return ErrStorage{Operation: op, Details: details}
}

func (e ErrStorage) Error() string {
	return fmt.Sprintf("ошибка хранилища при %s: %s", e.Operation, e.Details)
}
