package imageservice

import (
	"context"

	dto "github.com/Oleja123/estate-agency/internal/application/image/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/image"
)

type Service interface {
	Create(ctx context.Context, req dto.CreateImageRequest) (domain.PropertyImage, error)
	CreateMany(ctx context.Context, req dto.CreateImagesRequest) ([]domain.PropertyImage, error)
	GetByID(ctx context.Context, id int) (domain.PropertyImage, error)
	ListByProperty(ctx context.Context, propertyID int) ([]dto.ImageDTO, error)
	Delete(ctx context.Context, id int) error
}
