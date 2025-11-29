package imagedb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/Oleja123/estate-agency/internal/domain/image"
	"github.com/Oleja123/estate-agency/internal/infrastructure/basedb"
	"github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	"github.com/Oleja123/estate-agency/internal/infrastructure/client"
	"github.com/jackc/pgx/v5"
)

// ImageRepository provides DB operations for property images.
type ImageRepository struct {
	*basedb.BaseRepository
	sq squirrel.StatementBuilderType
}

func New(client client.Client, logger *slog.Logger) *ImageRepository {
	return &ImageRepository{
		BaseRepository: basedb.New(client, logger),
		sq:             squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *ImageRepository) Create(ctx context.Context, img image.PropertyImage) (int, error) {
	const op = "propertydb.ImageRepository.Create"

	sql, args, err := r.sq.
		Insert("property_images").
		Columns("property_id", "path", "created_at").
		Values(img.PropertyID, img.Path, time.Now()).
		Suffix("RETURNING id, created_at").
		ToSql()

	if err != nil {
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	r.Logger.DebugContext(ctx, "create property image", "operation", op, "property_id", img.PropertyID)

	err = r.Client.QueryRow(ctx, sql, args...).Scan(&img.ID, &img.CreatedAt)
	if err != nil {
		return 0, r.HandleError(op, err)
	}

	return img.ID, nil
}

// CreateMany inserts multiple images in a single query and returns their generated IDs.
func (r *ImageRepository) CreateMany(ctx context.Context, imgs []image.PropertyImage) ([]int, error) {
	const op = "propertydb.ImageRepository.CreateMany"

	if len(imgs) == 0 {
		return nil, nil
	}

	now := time.Now()
	insert := r.sq.Insert("property_images").Columns("property_id", "path", "created_at")
	for _, img := range imgs {
		insert = insert.Values(img.PropertyID, img.Path, now)
	}
	sql, args, err := insert.Suffix("RETURNING id, created_at").ToSql()
	if err != nil {
		return nil, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	rows, err := r.Client.Query(ctx, sql, args...)
	if err != nil {
		return nil, r.HandleError(op, err)
	}
	defer rows.Close()

	ids := make([]int, 0, len(imgs))
	var createdAt time.Time
	for rows.Next() {
		var id int
		if err := rows.Scan(&id, &createdAt); err != nil {
			return nil, fmt.Errorf("scan returned id error: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return ids, nil
}

func (r *ImageRepository) GetByID(ctx context.Context, id int) (image.PropertyImage, error) {
	const op = "propertydb.ImageRepository.GetByID"

	sql, args, err := r.sq.
		Select("*").
		From("property_images").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return image.PropertyImage{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	img, err := r.scanImage(r.Client.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return image.PropertyImage{}, basedberrors.NewErrNotFound("property_image", id)
		}
		return image.PropertyImage{}, r.HandleError(op, err)
	}

	return img, nil
}

func (r *ImageRepository) ListByProperty(ctx context.Context, propertyID int) ([]image.PropertyImage, error) {
	const op = "propertydb.ImageRepository.ListByProperty"

	sql, args, err := r.sq.
		Select("*").
		From("property_images").
		Where(squirrel.Eq{"property_id": propertyID}).
		OrderBy("created_at DESC").
		ToSql()

	if err != nil {
		return nil, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	rows, err := r.Client.Query(ctx, sql, args...)
	if err != nil {
		return nil, r.HandleError(op, err)
	}
	defer rows.Close()

	var imgs []image.PropertyImage
	for rows.Next() {
		img, err := r.scanImage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		imgs = append(imgs, img)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return imgs, nil
}

func (r *ImageRepository) Delete(ctx context.Context, id int) (int, error) {
	const op = "propertydb.ImageRepository.Delete"

	sql, args, err := r.sq.
		Delete("property_images").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return 0, r.HandleError(op, err)
	}

	if result.RowsAffected() == 0 {
		return 0, basedberrors.NewErrNotFound("property_image", id)
	}

	return id, nil
}

// DeleteMany removes multiple images by their IDs in a single query.
func (r *ImageRepository) DeleteMany(ctx context.Context, propertyID int) ([]int, error) {
	const op = "propertydb.ImageRepository.DeleteMany"
	if propertyID == 0 {
		return nil, basedberrors.NewErrInvalidInput("property_id", propertyID, "invalid or zero id")
	}

	// Ensure property exists — deleting images for a non-existent property should return NotFound.
	var existsInt int
	checkSql, checkArgs, err := r.sq.Select("1").From("properties").Where(squirrel.Eq{"id": propertyID}).Limit(1).ToSql()
	if err != nil {
		return nil, basedberrors.NewErrDatabase(op, fmt.Sprintf("build property check query: %s", err))
	}
	row := r.Client.QueryRow(ctx, checkSql, checkArgs...)
	if err := row.Scan(&existsInt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, basedberrors.NewErrNotFound("property", propertyID)
		}
		return nil, r.HandleError(op, err)
	}

	// Delete rows and RETURNING id so we can return deleted ids to caller.
	sql, args, err := r.sq.
		Delete("property_images").
		Where(squirrel.Eq{"property_id": propertyID}).
		Suffix("RETURNING id").
		ToSql()

	if err != nil {
		return nil, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	rows, err := r.Client.Query(ctx, sql, args...)
	if err != nil {
		return nil, r.HandleError(op, err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan returned id error: %w", err)
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// deletion of zero rows is acceptable (no images existed) — return empty slice.
	return ids, nil
}

// scanImage reads a single property_images row into the domain model
func (r *ImageRepository) scanImage(sc basedb.RowScanner) (image.PropertyImage, error) {
	var img image.PropertyImage
	if err := sc.Scan(&img.ID, &img.PropertyID, &img.Path, &img.CreatedAt); err != nil {
		return image.PropertyImage{}, err
	}
	return img, nil
}
