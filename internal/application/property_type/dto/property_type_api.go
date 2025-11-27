package dto

import "time"

// PropertyTypeDTO is the API representation of a property type used in OpenAPI docs.
type PropertyTypeDTO struct {
	Id        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
