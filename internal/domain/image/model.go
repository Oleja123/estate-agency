package image

import "time"

type PropertyImage struct {
	ID         int       `json:"id"`
	PropertyID int       `json:"property_id"`
	Path       string    `json:"path"`
	CreatedAt  time.Time `json:"created_at"`
}
