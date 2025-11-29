package dto

import (
	property "github.com/Oleja123/estate-agency/internal/domain/property"
	optional "github.com/denpa16/optional-go-type"
)

type CreatePropertyRequest struct {
	Title               string  `json:"title"`
	PropertyDescription string  `json:"property_description"`
	TypeID              int     `json:"type_id"`
	TransactionType     string  `json:"transaction_type"`
	Price               float64 `json:"price"`
	Area                float64 `json:"area"`
	PropertyAddress     string  `json:"property_address"`
	City                string  `json:"city"`
}

type UpdatePropertyRequest struct {
	ID                  int                      `json:"-"`
	Title               optional.OptionalString  `json:"title,omitempty"`
	PropertyDescription optional.OptionalString  `json:"property_description,omitempty"`
	TypeID              optional.OptionalInt     `json:"type_id,omitempty"`
	TransactionType     optional.OptionalString  `json:"transaction_type,omitempty"`
	Price               optional.OptionalFloat64 `json:"price,omitempty"`
	Area                optional.OptionalFloat64 `json:"area,omitempty"`
	PropertyAddress     optional.OptionalString  `json:"property_address,omitempty"`
	City                optional.OptionalString  `json:"city,omitempty"`
	PropertyStatus      optional.OptionalString  `json:"property_status,omitempty"`
}

type ListPropertiesRequest struct {
	Filter property.Filter `json:"filter"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type ListPropertiesResponse struct {
	Properties []property.Property `json:"properties"`
	Total      int           `json:"total"`
}
