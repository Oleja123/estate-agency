package propertytypehandler

// Documentation-only DTOs for property type endpoints to simplify swagger parsing.
type PropertyTypeDTODoc struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type ListPropertyTypesResponseDoc struct {
	Types []PropertyTypeDTODoc `json:"types"`
	Total int                  `json:"total"`
}

type CreatePropertyTypeDoc struct {
	Name string `json:"name"`
}

type UpdatePropertyTypeDoc struct {
	Name string `json:"name"`
}
