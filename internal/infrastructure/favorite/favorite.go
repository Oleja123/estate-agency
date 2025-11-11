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
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("запрос: %s", err))
	}

	r.Logger.DebugContext(ctx, "добавление в избранное",
		"операция", op,
		"user_id", fav.UserID,
		"property_id", fav.PropertyID,
	)

	_, err = r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "добавлено в избранное",
		"операция", op,
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
		return favorite.Favorite{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "поиск избранного по пользователю и property",
		"операция", op,
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
			r.Logger.DebugContext(ctx, "избранное не найдено",
				"операция", op,
				"user_id", userID,
				"property_id", propertyID,
			)
			return favorite.Favorite{}, basedberrors.NewErrNotFound("favorite", fmt.Sprintf("user:%d,property:%d", userID, propertyID))
		}
		return favorite.Favorite{}, r.HandleError(op, err)
	}

	r.Logger.DebugContext(ctx, "избранное найдено",
		"операция", op,
		"user_id", userID,
		"property_id", propertyID,
	)

	return fav, nil
}

func (r *Repository) Delete(ctx context.Context, userID, propertyID int) error {
	const op = "favoritedb.Repository.Delete"

	sql, args, err := r.sq.
		Delete("favorites").
		Where(squirrel.Eq{"user_id": userID, "property_id": propertyID}).
		ToSql()

	if err != nil {
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "удаление из избранного",
		"операция", op,
		"user_id", userID,
		"property_id", propertyID,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return r.HandleError(op, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.Logger.DebugContext(ctx, "избранное не найдено для удаления",
			"операция", op,
			"user_id", userID,
			"property_id", propertyID,
		)
		return basedberrors.NewErrNotFound("favorite", fmt.Sprintf("user:%d,property:%d", userID, propertyID))
	}

	r.Logger.InfoContext(ctx, "удалено из избранного",
		"операция", op,
		"user_id", userID,
		"property_id", propertyID,
		"rows_affected", rowsAffected,
	)

	return nil
}

func (r *Repository) List(ctx context.Context, req favorite.ListRequest) ([]favorite.Favorite, error) {
	const op = "favoritedb.Repository.List"

	var favorites []favorite.Favorite

	query := r.sq.
		Select("*").
		From("favorites")

	query = r.applyFilters(query, req.Filter)

	sql, args, err := query.
		OrderBy("created_at DESC").
		Limit(uint64(req.Limit)).
		Offset(uint64(req.Offset)).
		ToSql()

	if err != nil {
		return nil, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	r.Logger.DebugContext(ctx, "получение списка избранного",
		"операция", op,
		"limit", req.Limit,
		"offset", req.Offset,
		"фильтры", fmt.Sprintf("%+v", req.Filter),
	)

	rows, err := r.Client.Query(ctx, sql, args...)
	if err != nil {
		return nil, r.HandleError(op, err)
	}
	defer rows.Close()

	for rows.Next() {
		var fav favorite.Favorite
		err := rows.Scan(
			&fav.UserID,
			&fav.PropertyID,
			&fav.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		favorites = append(favorites, fav)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
	}

	r.Logger.DebugContext(ctx, "список избранного получен",
		"операция", op,
		"favorites_count", len(favorites),
	)

	return favorites, nil
}

func (r *Repository) Exists(ctx context.Context, userID, propertyID int) (bool, error) {
	const op = "favoritedb.Repository.Exists"

	sql, args, err := r.sq.
		Select("1").
		From("favorites").
		Where(squirrel.Eq{"user_id": userID, "property_id": propertyID}).
		ToSql()

	if err != nil {
		return false, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
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

func (r *Repository) applyFilters(query squirrel.SelectBuilder, filter favorite.Filter) squirrel.SelectBuilder {
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
