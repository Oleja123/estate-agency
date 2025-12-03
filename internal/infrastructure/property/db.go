package propertydb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/Oleja123/estate-agency/internal/domain/property"
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

func (r *Repository) Create(ctx context.Context, prop property.Property) (int, error) {
	const op = "propertydb.Repository.Create"

	sql, args, err := r.sq.
		Insert("properties").
		Columns(
			"title", "property_description", "type_id", "transaction_type",
			"price", "area", "property_address", "latitude", "longitude",
			"city", "property_status", "created_by", "created_at", "updated_at",
		).
		Values(
			prop.Title, prop.PropertyDescription, prop.TypeID, prop.TransactionType,
			prop.Price, prop.Area, prop.PropertyAddress, prop.Latitude, prop.Longitude,
			prop.City, prop.PropertyStatus, prop.CreatedBy, time.Now(), time.Now(),
		).
		Suffix("RETURNING id, created_at, updated_at").
		ToSql()

	if err != nil {
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "creating property",
		"operation", op,
		"title", prop.Title,
		"transaction_type", prop.TransactionType,
	)

	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&prop.ID, &prop.CreatedAt, &prop.UpdatedAt,
	)

	if err != nil {
		return 0, r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "property created successfully",
		"operation", op,
		"property_id", prop.ID,
		"title", prop.Title,
	)

	return prop.ID, nil
}

func (r *Repository) GetByID(ctx context.Context, id int) (property.Property, error) {
	const op = "propertydb.Repository.GetByID"

	sql, args, err := r.sq.
		Select("*").
		From("properties").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return property.Property{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "searching property by id",
		"operation", op,
		"property_id", id,
	)

	var prop property.Property

	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&prop.ID, &prop.Title, &prop.PropertyDescription, &prop.TypeID,
		&prop.TransactionType, &prop.Price, &prop.Area, &prop.PropertyAddress,
		&prop.Latitude, &prop.Longitude, &prop.City, &prop.PropertyStatus,
		&prop.CreatedBy, &prop.CreatedAt, &prop.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "property not found by id",
				"operation", op,
				"property_id", id,
			)
			return property.Property{}, basedberrors.NewErrNotFound("property", id)
		}
		return property.Property{}, r.HandleError(op, err)
	}

	r.Logger.DebugContext(ctx, "property retrieved by id",
		"operation", op,
		"property_id", id,
		"title", prop.Title,
	)

	return prop, nil
}

// GetByIDWithFavorite returns the property and whether the specified user has it in favorites.
func (r *Repository) GetByIDWithFavorite(ctx context.Context, id int, userID int) (property.Property, bool, error) {
	const op = "propertydb.Repository.GetByIDWithFavorite"

	// Build the query using squirrel: select property columns and a boolean
	// computed by checking if a favorite row exists for the given user.
	// Use LEFT JOIN with the user filter in the join to avoid duplicating rows.
	sql, args, err := r.sq.
		Select(
			"p.id", "p.title", "p.property_description", "p.type_id", "p.transaction_type", "p.price", "p.area", "p.property_address", "p.latitude", "p.longitude", "p.city", "p.property_status", "p.created_by", "p.created_at", "p.updated_at", "COALESCE(f.user_id IS NOT NULL, false) AS is_favorited",
		).From("properties p").LeftJoin("favorites f ON f.property_id = p.id AND f.user_id = ?", userID).
		Where(squirrel.Eq{"p.id": id}).Limit(1).ToSql()
	if err != nil {
		return property.Property{}, false, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query error: %s", err))
	}

	r.Logger.DebugContext(ctx, "GetByIDWithFavorite query",
		"operation", op,
		"property_id", id,
		"user_id", userID,
	)

	var prop property.Property
	var isFav bool

	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&prop.ID, &prop.Title, &prop.PropertyDescription, &prop.TypeID,
		&prop.TransactionType, &prop.Price, &prop.Area, &prop.PropertyAddress,
		&prop.Latitude, &prop.Longitude, &prop.City, &prop.PropertyStatus,
		&prop.CreatedBy, &prop.CreatedAt, &prop.UpdatedAt, &isFav,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "property not found by id",
				"operation", op,
				"property_id", id,
			)
			return property.Property{}, false, basedberrors.NewErrNotFound("property", id)
		}
		return property.Property{}, false, r.HandleError(op, err)
	}

	return prop, isFav, nil
}

func (r *Repository) Update(ctx context.Context, prop property.Property) (property.Property, error) {
	const op = "propertydb.Repository.Update"

	sql, args, err := r.sq.
		Update("properties").
		Set("title", prop.Title).
		Set("property_description", prop.PropertyDescription).
		Set("type_id", prop.TypeID).
		Set("transaction_type", prop.TransactionType).
		Set("price", prop.Price).
		Set("area", prop.Area).
		Set("property_address", prop.PropertyAddress).
		Set("latitude", prop.Latitude).
		Set("longitude", prop.Longitude).
		Set("city", prop.City).
		Set("property_status", prop.PropertyStatus).
		Set("updated_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": prop.ID}).
		Suffix("RETURNING id, title, property_description, type_id, transaction_type, price, area, property_address, latitude, longitude, city, property_status, created_by, created_at, updated_at").
		ToSql()

	if err != nil {
		return property.Property{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "updating property",
		"operation", op,
		"property_id", prop.ID,
	)

	var updated property.Property
	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&updated.ID, &updated.Title, &updated.PropertyDescription, &updated.TypeID,
		&updated.TransactionType, &updated.Price, &updated.Area, &updated.PropertyAddress,
		&updated.Latitude, &updated.Longitude, &updated.City, &updated.PropertyStatus,
		&updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "property not found on update",
				"operation", op,
				"property_id", prop.ID,
			)
			return property.Property{}, basedberrors.NewErrNotFound("property", prop.ID)
		}
		return property.Property{}, r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "property updated successfully",
		"operation", op,
		"property_id", updated.ID,
	)

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int) (int, error) {
	const op = "propertydb.Repository.Delete"

	sql, args, err := r.sq.
		Delete("properties").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "deleting property",
		"operation", op,
		"property_id", id,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return 0, r.HandleError(op, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.Logger.DebugContext(ctx, "property not found",
			"operation", op,
			"property_id", id,
		)
		return 0, basedberrors.NewErrNotFound("property", id)
	}

	r.Logger.InfoContext(ctx, "property deleted successfully",
		"operation", op,
		"property_id", id,
		"rows_deleted", rowsAffected,
	)

	return id, nil
}

func (r *Repository) List(ctx context.Context, req property.ListRequest) ([]property.Property, int, error) {
	const op = "propertydb.Repository.List"

	var properties []property.Property

	query := r.sq.
		Select("*").
		From("properties")

	query = r.ApplyFilters(query, req.Filter)

	countQ := r.sq.Select("COUNT(1)").From("properties")
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

	r.Logger.DebugContext(ctx, "listing properties",
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
		prop, err := r.ScanProperty(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("row scan error: %w", err)
		}
		properties = append(properties, prop)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	r.Logger.DebugContext(ctx, "properties list retrieved",
		"operation", op,
		"count", len(properties),
	)

	return properties, total, nil
}

func (r *Repository) ScanProperty(sc basedb.RowScanner) (property.Property, error) {
	var prop property.Property
	if err := sc.Scan(
		&prop.ID, &prop.Title, &prop.PropertyDescription, &prop.TypeID,
		&prop.TransactionType, &prop.Price, &prop.Area, &prop.PropertyAddress,
		&prop.Latitude, &prop.Longitude, &prop.City, &prop.PropertyStatus,
		&prop.CreatedBy, &prop.CreatedAt, &prop.UpdatedAt,
	); err != nil {
		return property.Property{}, err
	}
	return prop, nil
}

func (r *Repository) ApplyFilters(query squirrel.SelectBuilder, filter property.Filter) squirrel.SelectBuilder {
	if len(filter.IDs) > 0 {
		query = query.Where(squirrel.Eq{"id": filter.IDs})
	}

	if len(filter.TypeIDs) > 0 {
		query = query.Where(squirrel.Eq{"type_id": filter.TypeIDs})
	}

	if filter.TransactionType != "" {
		query = query.Where(squirrel.Eq{"transaction_type": filter.TransactionType})
	}

	if filter.City != "" {
		query = query.Where(squirrel.Eq{"city": filter.City})
	}

	if filter.PropertyStatus != "" {
		query = query.Where(squirrel.Eq{"property_status": filter.PropertyStatus})
	}

	if filter.CreatedBy > 0 {
		query = query.Where(squirrel.Eq{"created_by": filter.CreatedBy})
	}

	if filter.MinPrice > 0 {
		query = query.Where(squirrel.GtOrEq{"price": filter.MinPrice})
	}

	if filter.MaxPrice > 0 {
		query = query.Where(squirrel.LtOrEq{"price": filter.MaxPrice})
	}

	if filter.MinArea > 0 {
		query = query.Where(squirrel.GtOrEq{"area": filter.MinArea})
	}

	if filter.MaxArea > 0 {
		query = query.Where(squirrel.LtOrEq{"area": filter.MaxArea})
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where(squirrel.Or{
			squirrel.ILike{"title": searchPattern},
			squirrel.ILike{"property_description": searchPattern},
			squirrel.ILike{"property_address": searchPattern},
			squirrel.ILike{"city": searchPattern},
		})
	}

	if filter.Latitude != 0 && filter.Longitude != 0 && filter.RadiusKm > 0 {
		distanceCondition := fmt.Sprintf(
			"ST_DWithin(ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography, ST_SetSRID(ST_MakePoint(%f, %f), 4326)::geography, %f)",
			filter.Longitude, filter.Latitude, filter.RadiusKm*1000,
		)

		query = query.Where(distanceCondition)
	}

	return query
}
