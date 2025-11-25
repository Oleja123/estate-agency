package propertydb

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

func NewImageRepository(client client.Client, logger *slog.Logger) *ImageRepository {
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

	var img image.PropertyImage
	err = r.Client.QueryRow(ctx, sql, args...).Scan(&img.ID, &img.PropertyID, &img.Path, &img.CreatedAt)
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
		var img image.PropertyImage
		if err := rows.Scan(&img.ID, &img.PropertyID, &img.Path, &img.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		imgs = append(imgs, img)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return imgs, nil
}

func (r *ImageRepository) Delete(ctx context.Context, id int) error {
	const op = "propertydb.ImageRepository.Delete"

	sql, args, err := r.sq.
		Delete("property_images").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return r.HandleError(op, err)
	}

	if result.RowsAffected() == 0 {
		return basedberrors.NewErrNotFound("property_image", id)
	}

	return nil
}
