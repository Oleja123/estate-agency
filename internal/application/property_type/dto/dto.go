package dto

// DTOs for application-level property type operations.
type CreateRequest struct {
	Name string `json:"name"`
}

type UpdateRequest struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
