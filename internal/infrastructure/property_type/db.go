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
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "creating property type",
		"operation", op,
		"name", propertyType.Name,
	)

	var Id int
	err = r.Client.QueryRow(ctx, sql, args...).Scan(&Id)
	if err != nil {
		return 0, r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "тип недвижимости успешно создан",
		"операция", op,
		"id типа", Id,
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
		return propertytype.PropertyType{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "getting property type by id",
		"operation", op,
		"type_id", Id,
	)

	// use shared scanner
	row := r.Client.QueryRow(ctx, sql, args...)
	pt, err := r.scanPropertyType(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "property type not found by id",
				"operation", op,
				"type_id", Id,
			)
			return propertytype.PropertyType{}, basedberrors.NewErrNotFound("property_type", Id)
		}
		return propertytype.PropertyType{}, r.HandleError(op, err)
	}

	r.Logger.DebugContext(ctx, "property type retrieved by id",
		"operation", op,
		"type_id", Id,
		"name", pt.Name,
	)

	return pt, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (propertytype.PropertyType, error) {
	const op = "propertytypedb.Repository.GetByName"

	sql, args, err := r.sq.
		Select("Id", "property_name", "created_at").
		From("property_types").
		Where(squirrel.Eq{"property_name": name}).
		ToSql()

	if err != nil {
		return propertytype.PropertyType{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "getting property type by name",
		"operation", op,
		"name", name,
	)

	row := r.Client.QueryRow(ctx, sql, args...)
	pt, err := r.scanPropertyType(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "property type not found by name",
				"operation", op,
				"name", name,
			)
			return propertytype.PropertyType{}, basedberrors.NewErrNotFound("property_type", name)
		}
		return propertytype.PropertyType{}, r.HandleError(op, err)
	}

	return pt, nil
}

func (r *Repository) Update(ctx context.Context, propertyType propertytype.PropertyType) error {
	const op = "propertytypedb.Repository.Update"

	sql, args, err := r.sq.
		Update("property_types").
		Set("property_name", propertyType.Name).
		Where(squirrel.Eq{"Id": propertyType.Id}).
		ToSql()

	if err != nil {
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "updating property type",
		"operation", op,
		"type_id", propertyType.Id,
		"new_name", propertyType.Name,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return r.HandleError(op, err)
	}

	if result.RowsAffected() == 0 {
		r.Logger.DebugContext(ctx, "property type not found for update",
			"operation", op,
			"type_id", propertyType.Id,
		)
		return basedberrors.NewErrNotFound("property_type", propertyType.Id)
	}

	r.Logger.InfoContext(ctx, "property type updated successfully",
		"operation", op,
		"type_id", propertyType.Id,
	)

	return nil
}

func (r *Repository) Delete(ctx context.Context, Id int) (int, error) {
	const op = "propertytypedb.Repository.Delete"

	sql, args, err := r.sq.
		Delete("property_types").
		Where(squirrel.Eq{"Id": Id}).
		ToSql()

	if err != nil {
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "deleting property type",
		"operation", op,
		"type_id", Id,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return 0, r.HandleError(op, err)
	}

	if result.RowsAffected() == 0 {
		r.Logger.DebugContext(ctx, "property type not found for deletion",
			"operation", op,
			"type_id", Id,
		)
		return 0, basedberrors.NewErrNotFound("property_type", Id)
	}

	r.Logger.InfoContext(ctx, "property type deleted successfully",
		"operation", op,
		"type_id", Id,
		"rows_deleted", result.RowsAffected(),
	)

	return Id, nil
}

func (r *Repository) List(ctx context.Context, req propertytype.ListRequest) ([]propertytype.PropertyType, int, error) {
	const op = "propertytypedb.Repository.List"

	query := r.sq.
		Select("Id", "property_name", "created_at").
		From("property_types")

	query = r.applyFilters(query, req.Filter)

	// build count query to get total for pagination
	countQ := r.sq.Select("COUNT(1)").From("property_types")
	countQ = r.applyFilters(countQ, req.Filter)
	countSQL, countArgs, err := countQ.ToSql()
	if err != nil {
		return nil, 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка построения count запроса: %s", err))
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
		return nil, 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка построения запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "listing property types",
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

	var propertyTypes []propertytype.PropertyType
	for rows.Next() {
		pt, err := r.scanPropertyType(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("row scan error: %w", err)
		}
		propertyTypes = append(propertyTypes, pt)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	r.Logger.DebugContext(ctx, "property types list retrieved",
		"operation", op,
		"count", len(propertyTypes),
	)

	return propertyTypes, total, nil
}

// rowScanner abstracts types that support Scan(...interface{}) error
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanPropertyType reads a single property_type row into the domain model
func (r *Repository) scanPropertyType(sc rowScanner) (propertytype.PropertyType, error) {
	var pt propertytype.PropertyType
	if err := sc.Scan(&pt.Id, &pt.Name, &pt.CreatedAt); err != nil {
		return propertytype.PropertyType{}, err
	}
	return pt, nil
}

func (r *Repository) applyFilters(query squirrel.SelectBuilder, filter propertytype.Filter) squirrel.SelectBuilder {
	if len(filter.IDs) > 0 {
		query = query.Where(squirrel.Eq{"Id": filter.IDs})
	}

	if filter.Name != "" {
		query = query.Where(squirrel.Eq{"property_name": filter.Name})
	}

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where(squirrel.ILike{"property_name": searchPattern})
	}

	return query
}
