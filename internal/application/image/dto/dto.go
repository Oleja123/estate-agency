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

// ImageDTO represents images grouped by property. It contains the property id
// and a list of image files for that property.
type ImageDTO struct {
	PropertyID int         `json:"property_id"`
	Files      []ImageFile `json:"files"`
}

// CreatedAt omitted for simplicity; add time.Time if needed by consumers.
