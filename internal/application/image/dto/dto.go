package dto

// DTOs for image service
type ImageFile struct {
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}

type CreateImagesRequest struct {
	PropertyID int         `json:"property_id"`
	Files      []ImageFile `json:"files"`
}

type CreateImageRequest struct {
	PropertyID int       `json:"property_id"`
	File       ImageFile `json:"file"`
}

// ImageDTO is the representation returned to callers for images.
type ImageDTO struct {
	ID         int    `json:"id"`
	PropertyID int    `json:"property_id"`
	Filename   string `json:"filename"`
	Data       []byte `json:"data"`
	// CreatedAt omitted for simplicity; add time.Time if needed by consumers.
}
