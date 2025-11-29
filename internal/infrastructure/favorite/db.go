package favoritedb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/Oleja123/estate-agency/internal/domain/favorite"
	"github.com/Oleja123/estate-agency/internal/infrastructure/basedb"
	"github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	"github.com/Oleja123/estate-agency/internal/infrastructure/client"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	*basedb.BaseRepository
	sq squirrel.StatementBuilderType
}

func New(client client.Client, logger *slog.Logger) *Repository {
	return &Repository{
		BaseRepository: basedb.New(client, logger),
		sq:             squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *Repository) Create(ctx context.Context, fav favorite.Favorite) error {
	const op = "favoritedb.Repository.Create"

	sql, args, err := r.sq.
		Insert("favorites").
		Columns("user_id", "property_id").
		Values(fav.UserID, fav.PropertyID).
		ToSql()

	if err != nil {
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "adding to favorites",
		"operation", op,
		"user_id", fav.UserID,
		"property_id", fav.PropertyID,
	)

	_, err = r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "added to favorites",
		"operation", op,
		"user_id", fav.UserID,
		"property_id", fav.PropertyID,
	)

	return nil
}

func (r *Repository) GetByUserAndProperty(ctx context.Context, userID, propertyID int) (favorite.Favorite, error) {
	const op = "favoritedb.Repository.GetByUserAndProperty"

	sql, args, err := r.sq.
		Select("*").
		From("favorites").
		Where(squirrel.Eq{"user_id": userID, "property_id": propertyID}).
		ToSql()

	if err != nil {
		return favorite.Favorite{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "searching favorite by user and property",
		"operation", op,
		"user_id", userID,
		"property_id", propertyID,
	)

	var fav favorite.Favorite
	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&fav.UserID,
		&fav.PropertyID,
		&fav.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "favorite not found",
				"operation", op,
				"user_id", userID,
				"property_id", propertyID,
			)
			return favorite.Favorite{}, basedberrors.NewErrNotFound("favorite", fmt.Sprintf("user:%d,property:%d", userID, propertyID))
		}
		return favorite.Favorite{}, r.HandleError(op, err)
	}

	r.Logger.DebugContext(ctx, "favorite found",
		"operation", op,
		"user_id", userID,
		"property_id", propertyID,
	)

	return fav, nil
}

func (r *Repository) Delete(ctx context.Context, userID, propertyID int) (int, error) {
	const op = "favoritedb.Repository.Delete"

	sql, args, err := r.sq.
		Delete("favorites").
		Where(squirrel.Eq{"user_id": userID, "property_id": propertyID}).
		ToSql()

	if err != nil {
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "deleting from favorites",
		"operation", op,
		"user_id", userID,
		"property_id", propertyID,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return 0, r.HandleError(op, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.Logger.DebugContext(ctx, "favorite not found for deletion",
			"operation", op,
			"user_id", userID,
			"property_id", propertyID,
		)
		return 0, basedberrors.NewErrNotFound("favorite", fmt.Sprintf("user:%d,property:%d", userID, propertyID))
	}

	r.Logger.InfoContext(ctx, "deleted from favorites",
		"operation", op,
		"user_id", userID,
		"property_id", propertyID,
		"rows_affected", rowsAffected,
	)

	return propertyID, nil
}

func (r *Repository) List(ctx context.Context, req favorite.ListRequest) ([]favorite.Favorite, int, error) {
	const op = "favoritedb.Repository.List"

	var favorites []favorite.Favorite

	query := r.sq.
		Select("*").
		From("favorites")

	query = r.ApplyFilters(query, req.Filter)

	countQ := r.sq.Select("COUNT(1)").From("favorites")
	countQ = r.ApplyFilters(countQ, req.Filter)
	countSQL, countArgs, err := countQ.ToSql()
	if err != nil {
		return nil, 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("build count query: %s", err))
	}

	var total int
	if err := r.Client.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, r.HandleError(op, err)
	}

	sql, args, err := query.
		OrderBy("created_at DESC").
		Limit(uint64(req.Limit)).
		Offset(uint64(req.Offset)).
		ToSql()

	if err != nil {
		return nil, 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	r.Logger.DebugContext(ctx, "listing favorites",
		"operation", op,
		"limit", req.Limit,
		"offset", req.Offset,
		"filters", fmt.Sprintf("%+v", req.Filter),
	)

	rows, err := r.Client.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, r.HandleError(op, err)
	}
	defer rows.Close()

	for rows.Next() {
		fav, err := r.ScanFavorite(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("row scan error: %w", err)
		}
		favorites = append(favorites, fav)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	r.Logger.DebugContext(ctx, "favorites list retrieved",
		"operation", op,
		"favorites_count", len(favorites),
	)

	return favorites, total, nil
}

func (r *Repository) Exists(ctx context.Context, userID, propertyID int) (bool, error) {
	const op = "favoritedb.Repository.Exists"

	sql, args, err := r.sq.
		Select("1").
		From("favorites").
		Where(squirrel.Eq{"user_id": userID, "property_id": propertyID}).
		ToSql()

	if err != nil {
		return false, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	var exists int
	err = r.Client.QueryRow(ctx, sql, args...).Scan(&exists)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, r.HandleError(op, err)
	}

	return true, nil
}

func (r *Repository) ApplyFilters(query squirrel.SelectBuilder, filter favorite.Filter) squirrel.SelectBuilder {
	if filter.UserID > 0 {
		query = query.Where(squirrel.Eq{"user_id": filter.UserID})
	}

	if filter.PropertyID > 0 {
		query = query.Where(squirrel.Eq{"property_id": filter.PropertyID})
	}

	if len(filter.UserIDs) > 0 {
		query = query.Where(squirrel.Eq{"user_id": filter.UserIDs})
	}

	if len(filter.PropertyIDs) > 0 {
		query = query.Where(squirrel.Eq{"property_id": filter.PropertyIDs})
	}

	return query
}

func (r *Repository) ScanFavorite(sc basedb.RowScanner) (favorite.Favorite, error) {
	var fav favorite.Favorite
	if err := sc.Scan(&fav.UserID, &fav.PropertyID, &fav.CreatedAt); err != nil {
		return favorite.Favorite{}, err
	}
	return fav, nil
}
