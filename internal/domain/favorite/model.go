package favorite

import "time"

type Favorite struct {
	UserID     int       `json:"user_id"`
	PropertyID int       `json:"property_id"`
	CreatedAt  time.Time `json:"created_at"`
}
