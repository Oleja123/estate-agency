package favoriteservice

import (
	"context"
	"errors"
	"log/slog"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/favorite"
	prop "github.com/Oleja123/estate-agency/internal/domain/property"
	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
)

var _ Service = (*service)(nil)

type service struct {
	repo   domain.Repository
	logger *slog.Logger
}

func New(repo domain.Repository, logger *slog.Logger) Service {
	return &service{repo: repo, logger: logger}
}

func (s *service) Create(ctx context.Context, req dto.CreateFavoriteRequest) (int, error) {
	if req.UserID == 0 {
		return 0, apperrors.NewErrInvalidInput("user_id", req.UserID, "must be provided")
	}
	if req.PropertyID == 0 {
		return 0, apperrors.NewErrInvalidInput("property_id", req.PropertyID, "must be provided")
	}

	fav := domain.Favorite{UserID: req.UserID, PropertyID: req.PropertyID}
	if err := s.repo.Create(ctx, fav); err != nil {
		var ae dberrors.ErrAlreadyExists
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &ae):
			return 0, apperrors.NewErrAlreadyExists("favorite", "user_id,property_id", nil)
		case errors.As(err, &te):
			s.logger.Error("create favorite: repo timeout", "err", err)
			return 0, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("create favorite: repo error", "err", err)
			return 0, apperrors.NewErrInternal("failed to create favorite")
		}
	}
	return req.PropertyID, nil
}

func (s *service) GetByUserAndProperty(ctx context.Context, key dto.CreateFavoriteRequest) (prop.Property, error) {
	p, err := s.repo.GetByUserAndProperty(ctx, key.UserID, key.PropertyID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return prop.Property{}, apperrors.NewErrNotFound("favorite", map[string]int{"user_id": key.UserID, "property_id": key.PropertyID})
		case errors.As(err, &te):
			s.logger.Error("get favorite: repo timeout", "err", err)
			return prop.Property{}, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("get favorite: repo error", "err", err)
			return prop.Property{}, apperrors.NewErrInternal("failed to fetch favorite")
		}
	}
	return p, nil
}

func (s *service) Delete(ctx context.Context, key dto.CreateFavoriteRequest) (int, error) {
	deletedID, err := s.repo.Delete(ctx, key.UserID, key.PropertyID)
	if err != nil {
		var nf dberrors.ErrNotFound
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &nf):
			return 0, apperrors.NewErrNotFound("favorite", map[string]int{"user_id": key.UserID, "property_id": key.PropertyID})
		case errors.As(err, &te):
			s.logger.Error("delete favorite: repo timeout", "err", err)
			return 0, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("delete favorite: repo error", "err", err)
			return 0, apperrors.NewErrInternal("failed to delete favorite")
		}
	}
	return deletedID, nil
}

func (s *service) List(ctx context.Context, req dto.ListFavoritesRequest) (dto.ListFavoritesResponse, error) {
	dr := domain.ListRequest{Filter: req.Filter, Limit: req.Limit, Offset: req.Offset}
	list, total, err := s.repo.List(ctx, dr)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("list favorites: repo timeout", "err", err)
			return dto.ListFavoritesResponse{}, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("list favorites: repo error", "err", err)
			return dto.ListFavoritesResponse{}, apperrors.NewErrInternal("failed to list favorites")
		}
	}
	return dto.ListFavoritesResponse{Favorites: list, Total: total}, nil
}

func (s *service) Exists(ctx context.Context, key dto.CreateFavoriteRequest) (bool, error) {
	ok, err := s.repo.Exists(ctx, key.UserID, key.PropertyID)
	if err != nil {
		var te dberrors.ErrTimeout
		switch {
		case errors.As(err, &te):
			s.logger.Error("exists favorite: repo timeout", "err", err)
			return false, apperrors.NewErrTimeout("request timeout")
		default:
			s.logger.Error("exists favorite: repo error", "err", err)
			return false, apperrors.NewErrInternal("failed to check favorite existence")
		}
	}
	return ok, nil
}
