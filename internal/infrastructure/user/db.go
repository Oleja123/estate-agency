package userdb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	sqlpkg "database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/Oleja123/estate-agency/internal/domain/user"
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

func (r *Repository) Create(ctx context.Context, user user.User) (int, error) {
	const op = "userdb.Repository.Create"

	sql, args, err := r.sq.
		Insert("users").
		Columns("email", "password_hash", "first_name", "last_name", "phone_number", "user_role", "created_at", "updated_at").
		Values(user.Email, user.PasswordHash, user.FirstName, user.LastName, user.PhoneNumber, user.UserRole, time.Now(), time.Now()).
		Suffix("RETURNING id, is_active, created_at, updated_at").
		ToSql()

	if err != nil {
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "creating user",
		"operation", op,
		"email", user.Email,
		"role", user.UserRole,
	)

	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&user.Id, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return 0, r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "user created successfully",
		"operation", op,
		"user_id", user.Id,
		"email", user.Email,
	)

	return user.Id, nil
}

func (r *Repository) GetByID(ctx context.Context, id int) (user.User, error) {
	const op = "userdb.Repository.GetByID"

	sql, args, err := r.sq.
		Select("*").
		From("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return user.User{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "searching user by id",
		"operation", op,
		"user_id", id,
	)

	u, err := r.ScanUser(r.Client.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "user not found by id",
				"operation", op,
				"user_id", id,
			)
			return user.User{}, basedberrors.NewErrNotFound("user", id)
		}
		return user.User{}, r.HandleError(op, err)
	}

	r.Logger.DebugContext(ctx, "user retrieved by id",
		"operation", op,
		"user_id", id,
		"email", u.Email,
	)

	return u, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (user.User, error) {
	const op = "userdb.Repository.GetByEmail"

	sql, args, err := r.sq.
		Select("*").
		From("users").
		Where(squirrel.Eq{"email": email}).
		ToSql()

	if err != nil {
		return user.User{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "searching user by email",
		"operation", op,
		"email", email,
	)

	u, err := r.ScanUser(r.Client.QueryRow(ctx, sql, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "user not found by email",
				"operation", op,
				"email", email,
			)
			return user.User{}, basedberrors.NewErrNotFound("user", email)
		}
		return user.User{}, r.HandleError(op, err)
	}

	return u, nil

}

func (r *Repository) List(ctx context.Context, req user.ListRequest) ([]user.User, int, error) {
	const op = "userdb.Repository.List"

	var users []user.User
	var total int

	query := r.sq.
		Select("*").
		From("users")

	query = r.ApplyFilters(query, req.Filter)

	sql, args, err := query.
		OrderBy("created_at DESC").
		Limit(uint64(req.Limit)).
		Offset(uint64(req.Offset)).
		ToSql()

	if err != nil {
		return nil, 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	r.Logger.DebugContext(ctx, "listing users",
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
		u, err := r.ScanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("row scan error: %w", err)
		}
		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	r.Logger.DebugContext(ctx, "users list retrieved",
		"operation", op,
		"count", len(users),
	)

	countQuery := r.sq.Select("COUNT(*)").From("users")
	countQuery = r.ApplyFilters(countQuery, req.Filter)
	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("build count query: %s", err))
	}
	if err := r.Client.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, r.HandleError(op, err)
	}

	return users, total, nil

}

func (r *Repository) Update(ctx context.Context, user user.User) error {
	const op = "userdb.Repository.Update"

	sql, args, err := r.sq.
		Update("users").
		Set("email", user.Email).
		Set("password_hash", user.PasswordHash).
		Set("first_name", user.FirstName).
		Set("last_name", user.LastName).
		Set("phone_number", user.PhoneNumber).
		Set("user_role", user.UserRole).
		Set("is_active", user.IsActive).
		Set("updated_at", squirrel.Expr("NOW()")).
		Where(squirrel.Eq{"id": user.Id}).
		Suffix("RETURNING updated_at").
		ToSql()

	if err != nil {
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "updating user",
		"operation", op,
		"user_id", user.Id,
	)

	err = r.Client.QueryRow(ctx, sql, args...).Scan(&user.UpdatedAt)

	if err != nil {
		return r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "user updated successfully",
		"operation", op,
		"user_id", user.Id,
	)

	return nil
}

func (r *Repository) Delete(ctx context.Context, id int) (int, error) {
	const op = "userdb.Repository.Delete"

	sql, args, err := r.sq.
		Delete("users").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("query build error: %s", err))
	}

	r.Logger.DebugContext(ctx, "deleting user",
		"operation", op,
		"user_id", id,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return 0, r.HandleError(op, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.Logger.DebugContext(ctx, "user not found for deletion",
			"operation", op,
			"user_id", id,
		)
		return 0, basedberrors.NewErrNotFound("user", id)
	}

	r.Logger.InfoContext(ctx, "user deleted successfully",
		"operation", op,
		"user_id", id,
		"rows_deleted", rowsAffected,
	)

	return id, nil
}

func (r *Repository) ApplyFilters(query squirrel.SelectBuilder, filter user.Filter) squirrel.SelectBuilder {
	if len(filter.IDs) > 0 {
		query = query.Where(squirrel.Eq{"id": filter.IDs})
	}

	if filter.Email != "" {
		query = query.Where(squirrel.Eq{"email": filter.Email})
	}

	if filter.UserRole != "" {
		query = query.Where(squirrel.Eq{"user_role": filter.UserRole})
	}

	if filter.IsActive != nil {
		query = query.Where(squirrel.Eq{"is_active": *filter.IsActive})
	}

	if filter.Search != "" {

		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where(squirrel.Or{
			squirrel.ILike{"first_name": searchPattern},
			squirrel.ILike{"last_name": searchPattern},
			squirrel.ILike{"email": searchPattern},
		})
	}

	return query
}

func (r *Repository) ScanUser(sc basedb.RowScanner) (user.User, error) {
	var u user.User
	var phone sqlpkg.NullString
	if err := sc.Scan(
		&u.Id, &u.Email, &u.PasswordHash, &u.FirstName,
		&u.LastName, &phone, &u.UserRole, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return user.User{}, err
	}
	if phone.Valid {
		u.PhoneNumber = &phone.String
	} else {
		u.PhoneNumber = nil
	}
	return u, nil
}
