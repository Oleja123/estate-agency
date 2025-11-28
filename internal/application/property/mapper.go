package propertyservice

import (
	"github.com/Oleja123/estate-agency/internal/application/property/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/property"
)

// MapProperty converts domain.Property to API DTO PropertyDTO.
func MapProperty(p domain.Property) dto.PropertyDTO {
	return dto.PropertyDTO{
		ID:                  p.ID,
		Title:               p.Title,
		PropertyDescription: p.PropertyDescription,
		TypeID:              p.TypeID,
		TransactionType:     string(p.TransactionType),
		Price:               p.Price,
		Area:                p.Area,
		PropertyAddress:     p.PropertyAddress,
		Latitude:            p.Latitude,
		Longitude:           p.Longitude,
		City:                p.City,
		PropertyStatus:      string(p.PropertyStatus),
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}
}

// MapProperties converts a slice of domain.Property to a slice of PropertyDTO.
func MapProperties(list []domain.Property) []dto.PropertyDTO {
	out := make([]dto.PropertyDTO, 0, len(list))
	for _, p := range list {
		out = append(out, MapProperty(p))
	}
	return out
}
