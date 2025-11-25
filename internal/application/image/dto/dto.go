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
    PropertyID int      `json:"property_id"`
    File       ImageFile `json:"file"`
}

