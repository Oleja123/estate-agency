package image

import "time"

// PropertyImage represents an image associated with a property.
// It stores the path on the server where the image file is located
// and references the Property by ID.
type PropertyImage struct {
	ID         int       `json:"id"`
	PropertyID int       `json:"property_id"`
	Path       string    `json:"path"` // server path to the image file
	CreatedAt  time.Time `json:"created_at"`
}
