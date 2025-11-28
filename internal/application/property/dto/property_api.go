package dto

import (
	"time"
)

// PropertyDTO is the API representation of a property used in OpenAPI docs.
type PropertyDTO struct {
	ID                  int       `json:"id"`
	Title               string    `json:"title"`
	PropertyDescription string    `json:"property_description"`
	TypeID              int       `json:"type_id"`
	TransactionType     string    `json:"transaction_type"`
	Price               float64   `json:"price"`
	Area                float64   `json:"area"`
	PropertyAddress     string    `json:"property_address"`
	Latitude            float64   `json:"latitude"`
	Longitude           float64   `json:"longitude"`
	City                string    `json:"city"`
	PropertyStatus      string    `json:"property_status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
