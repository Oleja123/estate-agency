package propertytypeservice

import (
	"time"
)

type PropertyType struct {
	Id        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
