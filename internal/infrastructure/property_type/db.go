package propertytypedb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Masterminds/squirrel"
	propertytype "github.com/Oleja123/estate-agency/internal/domain/property_type"
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

func (r *Repository) Create(ctx context.Context, propertyType propertytype.PropertyType) (int, error) {
	const op = "propertytypedb.Repository.Create"

	sql, args, err := r.sq.
		Insert("property_types").
		Columns("property_name", "created_at").
		Values(propertyType.Name, squirrel.Expr("NOW()")).
		Suffix("RETURNING Id").
		ToSql()

	if err != nil {
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка построения запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "создание типа недвижимости",
		"операция", op,
		"название", propertyType.Name,
	)

	var Id int
	err = r.Client.QueryRow(ctx, sql, args...).Scan(&Id)
	if err != nil {
		return 0, r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "тип недвижимости успешно создан",
		"операция", op,
		"id_типа", Id,
		"название", propertyType.Name,
	)

	return Id, nil
}

func (r *Repository) GetByID(ctx context.Context, Id int) (propertytype.PropertyType, error) {
	const op = "propertytypedb.Repository.GetByID"

	sql, args, err := r.sq.
		Select("Id", "property_name", "created_at").
		From("property_types").
		Where(squirrel.Eq{"Id": Id}).
		ToSql()

	if err != nil {
		return propertytype.PropertyType{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка построения запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "получение типа недвижимости по Id",
		"операция", op,
		"id_типа", Id,
	)

	var propertyType propertytype.PropertyType
	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&propertyType.Id,
		&propertyType.Name,
		&propertyType.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "тип недвижимости не найден по Id",
				"операция", op,
				"id_типа", Id,
			)
			return propertytype.PropertyType{}, basedberrors.NewErrNotFound("тип недвижимости", Id)
		}
		return propertytype.PropertyType{}, r.HandleError(op, err)
	}

	r.Logger.DebugContext(ctx, "тип недвижимости получен по Id",
		"операция", op,
		"id_типа", Id,
		"название", propertyType.Name,
	)

	return propertyType, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (propertytype.PropertyType, error) {
	const op = "propertytypedb.Repository.GetByName"

	sql, args, err := r.sq.
		Select("Id", "property_name", "created_at").
		From("property_types").
		Where(squirrel.Eq{"name": name}).
		ToSql()

	if err != nil {
		return propertytype.PropertyType{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка построения запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "получение типа недвижимости по названию",
		"операция", op,
		"название", name,
	)

	var propertyType propertytype.PropertyType
	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&propertyType.Id,
		&propertyType.Name,
		&propertyType.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "тип недвижимости не найден по названию",
				"операция", op,
				"название", name,
			)
			return propertytype.PropertyType{}, basedberrors.NewErrNotFound("тип недвижимости", name)
		}
		return propertytype.PropertyType{}, r.HandleError(op, err)
	}

	return propertyType, nil
}

func (r *Repository) Update(ctx context.Context, propertyType propertytype.PropertyType) error {
	const op = "propertytypedb.Repository.Update"

	sql, args, err := r.sq.
		Update("property_types").
		Set("property_name", propertyType.Name).
		Where(squirrel.Eq{"Id": propertyType.Id}).
		ToSql()

	if err != nil {
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка построения запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "обновление типа недвижимости",
		"операция", op,
		"id_типа", propertyType.Id,
		"новое_название", propertyType.Name,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return r.HandleError(op, err)
	}

	if result.RowsAffected() == 0 {
		r.Logger.DebugContext(ctx, "тип недвижимости не найден для обновления",
			"операция", op,
			"id_типа", propertyType.Id,
		)
		return basedberrors.NewErrNotFound("тип недвижимости", propertyType.Id)
	}

	r.Logger.InfoContext(ctx, "тип недвижимости успешно обновлен",
		"операция", op,
		"id_типа", propertyType.Id,
	)

	return nil
}

func (r *Repository) Delete(ctx context.Context, Id int) error {
	const op = "propertytypedb.Repository.Delete"

	sql, args, err := r.sq.
		Delete("property_types").
		Where(squirrel.Eq{"Id": Id}).
		ToSql()

	if err != nil {
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка построения запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "удаление типа недвижимости",
		"операция", op,
		"id_типа", Id,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return r.HandleError(op, err)
	}

	if result.RowsAffected() == 0 {
		r.Logger.DebugContext(ctx, "тип недвижимости не найден для удаления",
			"операция", op,
			"id_типа", Id,
		)
		return basedberrors.NewErrNotFound("тип недвижимости", Id)
	}

	r.Logger.InfoContext(ctx, "тип недвижимости успешно удален",
		"операция", op,
		"id_типа", Id,
	)

	return nil
}

func (r *Repository) List(ctx context.Context, req propertytype.ListRequest) ([]propertytype.PropertyType, error) {
	const op = "propertytypedb.Repository.List"

	query := r.sq.
		Select("Id", "property_name", "created_at").
		From("property_types")

	query = r.applyFilters(query, req.Filter)

	sql, args, err := query.
		OrderBy("created_at DESC").
		Limit(uint64(req.Limit)).
		Offset(uint64(req.Offset)).
		ToSql()

	if err != nil {
		return nil, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка построения запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "получение списка типов недвижимости",
		"операция", op,
		"лимит", req.Limit,
		"смещение", req.Offset,
		"фильтры", fmt.Sprintf("%+v", req.Filter),
	)

	rows, err := r.Client.Query(ctx, sql, args...)
	if err != nil {
		return nil, r.HandleError(op, err)
	}
	defer rows.Close()

	var propertyTypes []propertytype.PropertyType
	for rows.Next() {
		var pt propertytype.PropertyType
		err := rows.Scan(
			&pt.Id,
			&pt.Name,
			&pt.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		propertyTypes = append(propertyTypes, pt)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка обработки строк: %w", err)
	}

	r.Logger.DebugContext(ctx, "список типов недвижимости получен",
		"операция", op,
		"количество", len(propertyTypes),
	)

	return propertyTypes, nil
}

func (r *Repository) applyFilters(query squirrel.SelectBuilder, filter propertytype.Filter) squirrel.SelectBuilder {
	if len(filter.IDs) > 0 {
		query = query.Where(squirrel.Eq{"Id": filter.IDs})
	}

	if filter.Name != "" {
		query = query.Where(squirrel.Eq{"name": filter.Name})
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where(squirrel.ILike{"property_name": searchPattern})
	}

	return query
}
