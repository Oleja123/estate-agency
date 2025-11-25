package dto

import (
	propertytype "github.com/Oleja123/estate-agency/internal/domain/property_type"
)

// DTOs for application-level property type operations.
type CreatePropertyTypeRequest struct {
	Name string `json:"name"`
}

type UpdatePropertyTypeRequest struct {
	ID   int    `json:"-"`
	Name string `json:"name"`
}

type ListPropertyTypesRequest struct {
	Filter propertytype.Filter `json:"filter"`
	Limit  int                 `json:"limit"`
	Offset int                 `json:"offset"`
}

type ListPropertyTypesResponse struct {
	Types []propertytype.PropertyType `json:"types"`
	Total int                         `json:"total"`
}
