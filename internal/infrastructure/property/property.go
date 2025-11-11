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
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("запрос: %s", err))
	}

	r.Logger.DebugContext(ctx, "создание собственности",
		"операция", op,
		"название", prop.Title,
		"вид транзакции", prop.TransactionType,
	)

	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&prop.ID, &prop.CreatedAt, &prop.UpdatedAt,
	)

	if err != nil {
		return 0, r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "property был успешно создан",
		"операция", op,
		"id собственности", prop.ID,
		"название", prop.Title,
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
		return property.Property{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "поиск property с id",
		"операция", op,
		"id собственности", id,
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
			r.Logger.DebugContext(ctx, "property по Id не найден",
				"операция", op,
				"id собственности", id,
			)
			return property.Property{}, basedberrors.NewErrNotFound("собственность", id)
		}
		return property.Property{}, r.HandleError(op, err)
	}

	r.Logger.DebugContext(ctx, "сосбвтенность по Id получена",
		"операция", op,
		"id собственности", id,
		"название", prop.Title,
	)

	return prop, nil
}

func (r *Repository) Update(ctx context.Context, prop property.Property) error {
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
		Suffix("RETURNING updated_at").
		ToSql()

	if err != nil {
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "обновление собственности",
		"операция", op,
		"id собственности", prop.ID,
	)

	err = r.Client.QueryRow(ctx, sql, args...).Scan(&prop.UpdatedAt)

	if err != nil {
		return r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "собственность успешно обновлена",
		"операция", op,
		"id собственности", prop.ID,
	)

	return nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	const op = "propertydb.Repository.Delete"

	sql, args, err := r.sq.
		Delete("properties").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "удаление собственности",
		"операция", op,
		"id собственности", id,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return r.HandleError(op, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.Logger.DebugContext(ctx, "соьственность не найдена",
			"операция", op,
			"id собственности", id,
		)
		return basedberrors.NewErrNotFound("property", id)
	}

	r.Logger.InfoContext(ctx, "собственность успешно удалена",
		"операция", op,
		"id собственности", id,
		"удалено строк", rowsAffected,
	)

	return nil
}

func (r *Repository) List(ctx context.Context, req property.ListRequest) ([]property.Property, error) {
	const op = "propertydb.Repository.List"

	var properties []property.Property

	query := r.sq.
		Select("*").
		From("properties")

	query = r.applyFilters(query, req.Filter)

	sql, args, err := query.
		OrderBy("created_at DESC").
		Limit(uint64(req.Limit)).
		Offset(uint64(req.Offset)).
		ToSql()

	if err != nil {
		return nil, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	r.Logger.DebugContext(ctx, "получение списка собственности",
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
		var prop property.Property
		err := rows.Scan(
			&prop.ID, &prop.Title, &prop.PropertyDescription, &prop.TypeID,
			&prop.TransactionType, &prop.Price, &prop.Area, &prop.PropertyAddress,
			&prop.Latitude, &prop.Longitude, &prop.City, &prop.PropertyStatus,
			&prop.CreatedBy, &prop.CreatedAt, &prop.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		properties = append(properties, prop)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
	}

	r.Logger.DebugContext(ctx, "список собственности получен",
		"операция", op,
		"количество", len(properties),
	)

	return properties, nil
}

func (r *Repository) applyFilters(query squirrel.SelectBuilder, filter property.Filter) squirrel.SelectBuilder {
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
			filter.Longitude, filter.Latitude, filter.RadiusKm*1000, // конвертируем км в метры
		)

		query = query.Where(distanceCondition)
	}

	return query
}
