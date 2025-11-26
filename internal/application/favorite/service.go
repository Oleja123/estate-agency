package favoriteservice

import (
	"context"
	"errors"
	"log/slog"

	apperrors "github.com/Oleja123/estate-agency/internal/application/errors"
	dto "github.com/Oleja123/estate-agency/internal/application/favorite/dto"
	domain "github.com/Oleja123/estate-agency/internal/domain/favorite"
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

func (s *service) Create(ctx context.Context, req dto.CreateFavoriteRequest) (domain.Favorite, error) {
	if req.UserID == 0 {
		return domain.Favorite{}, apperrors.NewErrInvalidInput("user_id", req.UserID, "must be provided")
	}
	if req.PropertyID == 0 {
		return domain.Favorite{}, apperrors.NewErrInvalidInput("property_id", req.PropertyID, "must be provided")
	}

	fav := domain.Favorite{UserID: req.UserID, PropertyID: req.PropertyID}
	if err := s.repo.Create(ctx, fav); err != nil {
		var ae dberrors.ErrAlreadyExists
		if errors.As(err, &ae) {
			return domain.Favorite{}, apperrors.NewErrAlreadyExists("favorite", "user_id,property_id", nil)
		}
		s.logger.Error("create favorite: repo error", "err", err)
		return domain.Favorite{}, apperrors.NewErrInternal("failed to create favorite")
	}

	created, err := s.repo.GetByUserAndProperty(ctx, req.UserID, req.PropertyID)
	if err != nil {
		s.logger.Error("create favorite: fetch created failed", "err", err)
		return domain.Favorite{}, apperrors.NewErrInternal("failed to fetch created favorite")
	}
	return created, nil
}

func (s *service) GetByUserAndProperty(ctx context.Context, key dto.CreateFavoriteRequest) (domain.Favorite, error) {
	fav, err := s.repo.GetByUserAndProperty(ctx, key.UserID, key.PropertyID)
	if err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return domain.Favorite{}, apperrors.NewErrNotFound("favorite", map[string]int{"user_id": key.UserID, "property_id": key.PropertyID})
		}
		s.logger.Error("get favorite: repo error", "err", err)
		return domain.Favorite{}, apperrors.NewErrInternal("failed to fetch favorite")
	}
	return fav, nil
}

func (s *service) Delete(ctx context.Context, key dto.CreateFavoriteRequest) error {
	if err := s.repo.Delete(ctx, key.UserID, key.PropertyID); err != nil {
		var nf dberrors.ErrNotFound
		if errors.As(err, &nf) {
			return apperrors.NewErrNotFound("favorite", map[string]int{"user_id": key.UserID, "property_id": key.PropertyID})
		}
		s.logger.Error("delete favorite: repo error", "err", err)
		return apperrors.NewErrInternal("failed to delete favorite")
	}
	return nil
}

func (s *service) List(ctx context.Context, req dto.ListFavoritesRequest) (dto.ListFavoritesResponse, error) {
	dr := domain.ListRequest{Filter: req.Filter, Limit: req.Limit, Offset: req.Offset}
	list, err := s.repo.List(ctx, dr)
	if err != nil {
		s.logger.Error("list favorites: repo error", "err", err)
		return dto.ListFavoritesResponse{}, apperrors.NewErrInternal("failed to list favorites")
	}
	total := len(list)
	return dto.ListFavoritesResponse{Favorites: list, Total: total}, nil
}

func (s *service) Exists(ctx context.Context, key dto.CreateFavoriteRequest) (bool, error) {
	ok, err := s.repo.Exists(ctx, key.UserID, key.PropertyID)
	if err != nil {
		s.logger.Error("exists favorite: repo error", "err", err)
		return false, apperrors.NewErrInternal("failed to check favorite existence")
	}
	return ok, nil
}
