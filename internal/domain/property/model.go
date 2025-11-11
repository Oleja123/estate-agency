package property

import (
	"time"
)

type TransactionType string

const (
	TransactionSale TransactionType = "sale"
	TransactionRent TransactionType = "rent"
)

type PropertyStatus string

const (
	StatusActive   PropertyStatus = "active"
	StatusSold     PropertyStatus = "sold"
	StatusRented   PropertyStatus = "rented"
	StatusInactive PropertyStatus = "inactive"
)

type Property struct {
	ID                  int             `json:"id"`
	Title               string          `json:"title"`
	PropertyDescription string          `json:"property_description"`
	TypeID              int             `json:"type_id"`
	TransactionType     TransactionType `json:"transaction_type"`
	Price               float64         `json:"price"`
	Area                float64         `json:"area"`
	PropertyAddress     string          `json:"property_address"`
	Latitude            float64         `json:"latitude"`
	Longitude           float64         `json:"longitude"`
	City                string          `json:"city"`
	PropertyStatus      PropertyStatus  `json:"property_status"`
	CreatedBy           int             `json:"created_by"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}
