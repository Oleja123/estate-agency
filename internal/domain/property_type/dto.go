package propertytypeservice

// DTOs for property type service operations.
type CreateRequest struct {
	Name string `json:"name"`
}

type UpdateRequest struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}
