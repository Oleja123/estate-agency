package basedb

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	dberrors "github.com/Oleja123/estate-agency/internal/infrastructure/basedb/basedberrors"
	"github.com/Oleja123/estate-agency/internal/infrastructure/client"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type BaseRepository struct {
	Client client.Client
	Logger *slog.Logger
}

func New(client client.Client, logger *slog.Logger) *BaseRepository {
	if logger == nil {
		logger = slog.Default()
	}
	return &BaseRepository{
		Client: client,
		Logger: logger,
	}
}

func (r *BaseRepository) HandleError(op string, err error) error {
	if err == nil {
		return nil
	}

	r.Logger.ErrorContext(context.Background(), "ошибка операции базы данных",
		"операция", op,
		"ошибка", err.Error(),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		r.Logger.DebugContext(context.Background(), "сущность не найдена", "операция", op)
		return dberrors.NewErrNotFound("сущность", nil)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		r.Logger.WarnContext(context.Background(), "ошибка postgres",
			"операция", op,
			"pg_code", pgErr.Code,
			"pg_message", pgErr.Message,
			"ограничение", pgErr.ConstraintName,
		)

		switch pgErr.Code {
		case "23505":
			entity, field := r.parseConstraintName(pgErr.ConstraintName)
			return dberrors.NewErrAlreadyExists(entity, field, "unknown")

		case "23503":
			return dberrors.NewErrForeignKeyViolation(
				pgErr.TableName,
				pgErr.ConstraintName,
				pgErr.Detail,
			)

		case "23502":
			return dberrors.NewErrInvalidInput(
				pgErr.ColumnName,
				pgErr.Detail,
				"поле не может быть null",
			)

		case "23514":
			return dberrors.NewErrInvalidInput(
				pgErr.ColumnName,
				pgErr.Detail,
				pgErr.Message,
			)

		case "08000", "08003", "08006":
			return dberrors.NewErrConnection(pgErr.Message)
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		r.Logger.WarnContext(context.Background(), "операция превысила лимит ожидания", "операция", op)
		return dberrors.NewErrTimeout(op, "превышен лимит ожидания")
	}
	if errors.Is(err, context.Canceled) {
		r.Logger.WarnContext(context.Background(), "операция отменена", "операция", op)
		return dberrors.NewErrTimeout(op, "контекст отменён")
	}

	if r.isConnectionError(err) {
		r.Logger.ErrorContext(context.Background(), "ошибка подключения",
			"операция", op,
			"оштбка", err.Error(),
		)
		return dberrors.NewErrConnection(err.Error())
	}

	return dberrors.NewErrDatabase(op, err.Error())
}

func (r *BaseRepository) parseConstraintName(constraintName string) (string, string) {
	parts := strings.Split(constraintName, "_")
	if len(parts) >= 2 {
		entity := parts[0]
		field := parts[1]
		return entity, field
	}
	return "entity", "field"
}

func (r *BaseRepository) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	connectionErrors := []string{
		"connection refused",
		"connection reset",
		"network unreachable",
		"driver: bad connection",
		"broken pipe",
		"conn closed",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(strings.ToLower(errorStr), connErr) {
			return true
		}
	}
	return false
}

func IsNotFound(err error) bool {
	var notFoundErr dberrors.ErrNotFound
	return errors.As(err, &notFoundErr)
}

func IsAlreadyExists(err error) bool {
	var existsErr dberrors.ErrAlreadyExists
	return errors.As(err, &existsErr)
}

func IsInvalidInput(err error) bool {
	var invalidErr dberrors.ErrInvalidInput
	return errors.As(err, &invalidErr)
}
