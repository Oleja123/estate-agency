package propertyhandler

import "time"

// Documentation-only DTOs for property handlers (kept in a separate file
// so handler source is focused on request handling).
type PropertyCreateDoc struct {
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

type PropertyDTODoc struct {
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
	CreatedBy           int       `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ListPropertiesResponseDoc struct {
	Properties []PropertyDTODoc `json:"properties"`
	Total      int              `json:"total"`
}

type ImageDTODoc struct {
	PropertyID int            `json:"property_id"`
	Files      []ImageFileDoc `json:"files"`
}

type ImageFileDoc struct {
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}

// UpdatePropertyDoc represents the request body for updating a property.
type UpdatePropertyDoc struct {
	Title               string  `json:"title"`
	PropertyDescription string  `json:"property_description"`
	TypeID              int     `json:"type_id"`
	TransactionType     string  `json:"transaction_type"`
	Price               float64 `json:"price"`
	Area                float64 `json:"area"`
	PropertyAddress     string  `json:"property_address"`
	City                string  `json:"city"`
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
	PropertyStatus      string  `json:"property_status"`
}

// FavoriteDTODoc describes a favorite relation between a user and a property.
type FavoriteDTODoc struct {
	UserID     int    `json:"user_id"`
	PropertyID int    `json:"property_id"`
	CreatedAt  string `json:"created_at"`
}
