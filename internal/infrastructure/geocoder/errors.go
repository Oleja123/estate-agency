package geocoder

import "fmt"

type ErrGeoRequest struct {
	Details string
}

func NewErrGeoRequest(details string) error {
	return ErrGeoRequest{Details: details}
}

func (e ErrGeoRequest) Error() string {
	return fmt.Sprintf("ошибка геокодирования: %s", e.Details)
}

type ErrGeoNoResults struct {
	Address string
}

func NewErrGeoNoResults(address string) error {
	return ErrGeoNoResults{Address: address}
}

func (e ErrGeoNoResults) Error() string {
	return fmt.Sprintf("не найдено координат для адреса: %s", e.Address)
}

type ErrGeoConfig struct {
	Message string
}

func NewErrGeoConfig(message string) error {
	return ErrGeoConfig{Message: message}
}

func (e ErrGeoConfig) Error() string {
	return fmt.Sprintf("ошибка конфигурации геокодера: %s", e.Message)
}

type ErrGeoDecode struct {
	Details string
}

func NewErrGeoDecode(details string) error {
	return ErrGeoDecode{Details: details}
}

func (e ErrGeoDecode) Error() string {
	return fmt.Sprintf("не удалось разобрать ответ геокодера: %s", e.Details)
}
