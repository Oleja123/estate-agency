package image

import "context"

// ImageRepository defines storage operations for property images.
type ImageRepository interface {
	// Create stores a new PropertyImage and returns its ID.
	Create(ctx context.Context, img PropertyImage) (int, error)

	// CreateMany stores multiple PropertyImage records and returns their IDs in order.
	CreateMany(ctx context.Context, imgs []PropertyImage) ([]int, error)

	// GetByID returns image by its ID.
	GetByID(ctx context.Context, id int) (PropertyImage, error)

	// ListByProperty returns all images for given property ID.
	ListByProperty(ctx context.Context, propertyID int) ([]PropertyImage, error)

	// Delete removes image by its ID and returns deleted id.
	Delete(ctx context.Context, id int) (int, error)

	// DeleteMany removes all images for a given property ID and returns their IDs.
	DeleteMany(ctx context.Context, propertyID int) ([]int, error)
}
