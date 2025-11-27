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
	City                string    `json:"city"`
	CreatedBy           int       `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ListPropertiesResponseDoc struct {
	Properties []PropertyDTODoc `json:"properties"`
	Total      int              `json:"total"`
}

type ImageDTODoc struct {
	ID         int    `json:"id"`
	PropertyID int    `json:"property_id"`
	Filename   string `json:"filename"`
	Data       []byte `json:"data"`
}
