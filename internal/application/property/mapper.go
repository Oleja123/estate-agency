package propertyservice

import (
	"path/filepath"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	imagedto "github.com/Oleja123/estate-agency/internal/application/image/dto"
	"github.com/Oleja123/estate-agency/internal/application/property/dto"
	imagedomain "github.com/Oleja123/estate-agency/internal/domain/image"
	domain "github.com/Oleja123/estate-agency/internal/domain/property"
)

func (s *service) MapProperty(p domain.Property) (dto.PropertyDTO, error) {
	images, err := s.getFilesImages(p.Images)
	if err != nil {
		s.logger.Error("failed to get files images", "error", err)
		return dto.PropertyDTO{}, err
	}

	var image *imagedto.ImageFile

	if len(images) > 0 {
		image = &images[0]
	}

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
		Images:              images,
		Image:               image,
	}, nil
}

func (s *service) getFilesImages(images []imagedomain.PropertyImage) ([]imagedto.ImageFile, error) {
	var result []imagedto.ImageFile
	for _, img := range images {
		data, err := s.fileStore.Read(img.Path)
		if err != nil {
			s.logger.Error("failed to read image file", "path", img.Path, "error", err)
			return nil, apperrors.NewErrInternal("не удалось открыть файл изображения")
		}
		file := imagedto.ImageFile{
			Filename: filepath.Base(img.Path),
			Data:     data,
		}
		result = append(result, file)
	}
	return result, nil
}

func (s *service) MapPropertyWithFavorite(p domain.Property, isFav bool) (dto.PropertyDTO, error) {
	out, err := s.MapProperty(p)
	if err != nil {
		return dto.PropertyDTO{}, err
	}
	out.IsFavorited = isFav
	return out, nil
}

func (s *service) MapProperties(list []domain.Property) ([]dto.PropertyDTO, error) {
	out := make([]dto.PropertyDTO, 0, len(list))
	for _, p := range list {
		prop, err := s.MapProperty(p)
		if err != nil {
			return nil, err
		}
		out = append(out, prop)
	}
	return out, nil
}
