package image

import "context"

// ImageRepository defines storage operations for property images.
type ImageRepository interface {
	// Create stores a new PropertyImage and returns its ID.
	Create(ctx context.Context, img PropertyImage) (int, error)

	// GetByID returns image by its ID.
	GetByID(ctx context.Context, id int) (PropertyImage, error)

	// ListByProperty returns all images for given property ID.
	ListByProperty(ctx context.Context, propertyID int) ([]PropertyImage, error)

	// Delete removes image by its ID.
	Delete(ctx context.Context, id int) error
}
