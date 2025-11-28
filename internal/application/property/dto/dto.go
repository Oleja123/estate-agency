package dto

import (
	property "github.com/Oleja123/estate-agency/internal/domain/property"
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
	CreatedBy           int     `json:"created_by"`
}

type UpdatePropertyRequest struct {
	ID                  int      `json:"-"`
	Title               *string  `json:"title,omitempty"`
	PropertyDescription *string  `json:"property_description,omitempty"`
	TypeID              *int     `json:"type_id,omitempty"`
	TransactionType     *string  `json:"transaction_type,omitempty"`
	Price               *float64 `json:"price,omitempty"`
	Area                *float64 `json:"area,omitempty"`
	PropertyAddress     *string  `json:"property_address,omitempty"`
	City                *string  `json:"city,omitempty"`
	PropertyStatus      *string  `json:"property_status,omitempty"`
}

type ListPropertiesRequest struct {
	Filter property.Filter `json:"filter"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type ListPropertiesResponse struct {
	Properties []property.Property `json:"properties"`
	Total      int                 `json:"total"`
}
