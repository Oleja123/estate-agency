package filestore

import "fmt"

// ErrInvalidInput indicates a problem with the provided input (filename, data, id etc).
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

// ErrUnsupportedFormat indicates the uploaded file has an unsupported image format.
type ErrUnsupportedFormat struct {
	Filename string
	Detected string
}

func NewErrUnsupportedFormat(filename, detected string) error {
	return ErrUnsupportedFormat{Filename: filename, Detected: detected}
}

func (e ErrUnsupportedFormat) Error() string {
	return fmt.Sprintf("unsupported image format for file '%s': %s", e.Filename, e.Detected)
}

// ErrStorage indicates a low-level storage error (IO, permission, etc).
type ErrStorage struct {
	Operation string
	Details   string
}

func NewErrStorage(op, details string) error {
	return ErrStorage{Operation: op, Details: details}
}

func (e ErrStorage) Error() string {
	return fmt.Sprintf("storage error during %s: %s", e.Operation, e.Details)
}
