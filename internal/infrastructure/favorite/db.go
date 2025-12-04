package favoritedb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"time"

	"github.com/Masterminds/squirrel"
	"github.com/Oleja123/estate-agency/internal/domain/favorite"
	imagedomain "github.com/Oleja123/estate-agency/internal/domain/image"
	prop "github.com/Oleja123/estate-agency/internal/domain/property"
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

func (r *Repository) GetByUserAndProperty(ctx context.Context, userID, propertyID int) (prop.Property, error) {
	const op = "favoritedb.Repository.GetByUserAndProperty"

	sql, args, err := r.sq.
		Select("p.id, p.title, p.property_description, p.type_id, p.transaction_type, p.price, p.area, p.property_address, p.latitude, p.longitude, p.city, p.property_status, p.created_by, p.created_at, p.updated_at, i.id AS image_id, i.property_id AS image_property_id, i.path AS image_path, i.created_at AS image_created_at").
		From("favorites f").
		Join("properties p ON p.id = f.property_id").
		LeftJoin("LATERAL (SELECT id, property_id, path, created_at FROM property_images WHERE property_id = p.id ORDER BY id ASC LIMIT 1) i ON true").
		Where(squirrel.Eq{"f.user_id": userID, "f.property_id": propertyID}).
		ToSql()

	if err != nil {
		return prop.Property{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "searching favorite -> property by user and property",
		"operation", op,
		"user_id", userID,
		"property_id", propertyID,
	)

	row := r.Client.QueryRow(ctx, sql, args...)
	p, err := r.ScanFavoriteProperty(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "favorite/property not found",
				"operation", op,
				"user_id", userID,
				"property_id", propertyID,
			)
			return prop.Property{}, basedberrors.NewErrNotFound("favorite.property", fmt.Sprintf("user:%d,property:%d", userID, propertyID))
		}
		return prop.Property{}, r.HandleError(op, err)
	}

	r.Logger.DebugContext(ctx, "favorite -> property found",
		"operation", op,
		"user_id", userID,
		"property_id", propertyID,
		"title", p.Title,
	)

	return p, nil
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

func (r *Repository) List(ctx context.Context, req favorite.ListRequest) ([]prop.Property, int, error) {
	const op = "favoritedb.Repository.List"
	var properties []prop.Property

	query := r.sq.
		Select("p.id, p.title, p.property_description, p.type_id, p.transaction_type, p.price, p.area, p.property_address, p.latitude, p.longitude, p.city, p.property_status, p.created_by, p.created_at, p.updated_at, i.id AS image_id, i.property_id AS image_property_id, i.path AS image_path, i.created_at AS image_created_at").
		From("favorites f").
		Join("properties p ON p.id = f.property_id").
		LeftJoin("LATERAL (SELECT id, property_id, path, created_at FROM property_images WHERE property_id = p.id ORDER BY id ASC LIMIT 1) i ON true")

	query = r.ApplyFilters(query, req.Filter)

	countQ := r.sq.Select("COUNT(1)").From("favorites f")
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
		OrderBy("f.created_at DESC").
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
		p, err := r.ScanFavoriteProperty(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("row scan error: %w", err)
		}
		properties = append(properties, p)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	r.Logger.DebugContext(ctx, "favorites (properties) list retrieved",
		"operation", op,
		"properties_count", len(properties),
	)

	return properties, total, nil
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
		query = query.Where(squirrel.Eq{"f.user_id": filter.UserID})
	}

	if filter.PropertyID > 0 {
		query = query.Where(squirrel.Eq{"f.property_id": filter.PropertyID})
	}

	if len(filter.UserIDs) > 0 {
		query = query.Where(squirrel.Eq{"f.user_id": filter.UserIDs})
	}

	if len(filter.PropertyIDs) > 0 {
		query = query.Where(squirrel.Eq{"f.property_id": filter.PropertyIDs})
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

func (r *Repository) ScanFavoriteProperty(sc basedb.RowScanner) (prop.Property, error) {
	var p prop.Property
	var imageID *int
	var imagePropertyID *int
	var imagePath *string
	var imageCreatedAt *time.Time

	if err := sc.Scan(
		&p.ID, &p.Title, &p.PropertyDescription, &p.TypeID,
		&p.TransactionType, &p.Price, &p.Area, &p.PropertyAddress,
		&p.Latitude, &p.Longitude, &p.City, &p.PropertyStatus,
		&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt,
		&imageID, &imagePropertyID, &imagePath, &imageCreatedAt,
	); err != nil {
		return prop.Property{}, err
	}

	if imageID != nil {
		img := imagedomain.PropertyImage{ID: *imageID}
		if imagePropertyID != nil {
			img.PropertyID = *imagePropertyID
		}
		if imagePath != nil {
			img.Path = *imagePath
		}
		if imageCreatedAt != nil {
			img.CreatedAt = *imageCreatedAt
		}
		p.Images = []imagedomain.PropertyImage{img}
	}

	return p, nil
}
