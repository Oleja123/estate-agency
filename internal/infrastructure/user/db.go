package userdb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("запрос: %s", err))
	}

	r.Logger.DebugContext(ctx, "создание пользователя",
		"операция", op,
		"email", user.Email,
		"роль", user.UserRole,
	)

	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&user.Id, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return 0, r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "пользователь был успешно создан",
		"операция", op,
		"id пользователя", user.Id,
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
		return user.User{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "поиск пользователя с id",
		"операция", op,
		"id пользоваетеля", id,
	)

	var u user.User

	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&u.Id, &u.Email, &u.PasswordHash, &u.FirstName,
		&u.LastName, &u.PhoneNumber, &u.UserRole, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "пользователь по Id не найден",
				"операция", op,
				"id пользователя", id,
			)
			return user.User{}, basedberrors.NewErrNotFound("пользователь", id)
		}
		return user.User{}, r.HandleError(op, err)
	}

	r.Logger.DebugContext(ctx, "пользователь по Id получен",
		"операция", op,
		"id пользователя", id,
		"email", u.Email,
	)

	return u, nil
}

func (r Repository) GetByEmail(ctx context.Context, email string) (user.User, error) {
	const op = "userdb.Repository.GetByEmail"

	sql, args, err := r.sq.
		Select("*").
		From("users").
		Where(squirrel.Eq{"email": email}).
		ToSql()

	if err != nil {
		return user.User{}, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "поиск пользователя с email",
		"операция", op,
		"email", email,
	)

	var u user.User
	err = r.Client.QueryRow(ctx, sql, args...).Scan(
		&u.Id, &u.Email, &u.PasswordHash, &u.FirstName,
		&u.LastName, &u.PhoneNumber, &u.UserRole, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.Logger.DebugContext(ctx, "пользователь с указанным email не найден",
				"операция", op,
				"email", email,
			)
			return user.User{}, basedberrors.NewErrNotFound("пользователь", email)
		}
		return user.User{}, r.HandleError(op, err)
	}

	return u, nil

}

func (r *Repository) List(ctx context.Context, req user.ListRequest) ([]user.User, error) {
	const op = "userdb.Repository.List"

	var users []user.User

	query := r.sq.
		Select("*").
		From("users")

	query = r.applyFilters(query, req.Filter)

	sql, args, err := query.
		OrderBy("created_at DESC").
		Limit(uint64(req.Limit)).
		Offset(uint64(req.Offset)).
		ToSql()

	if err != nil {
		return nil, basedberrors.NewErrDatabase(op, fmt.Sprintf("build query: %s", err))
	}

	r.Logger.DebugContext(ctx, "получение списка пользователнй",
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
		var user user.User
		err := rows.Scan(
			&user.Id, &user.Email, &user.PasswordHash, &user.FirstName,
			&user.LastName, &user.PhoneNumber, &user.UserRole, &user.IsActive,
			&user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
	}

	r.Logger.DebugContext(ctx, "список пользователей получен",
		"операция", op,
		"количество", len(users),
	)

	return users, nil

}

func (r *Repository) Update(ctx context.Context, user user.User) error {
	const op = "userdb.Repository.Update"

	sql, args, err := r.sq.
		Update("users").
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
		return basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "обновление пользователя",
		"операция", op,
		"user_id", user.Id,
	)

	err = r.Client.QueryRow(ctx, sql, args...).Scan(&user.UpdatedAt)

	if err != nil {
		return r.HandleError(op, err)
	}

	r.Logger.InfoContext(ctx, "пользователь успешно обновлен",
		"операция", op,
		"id пользователя", user.Id,
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
		return 0, basedberrors.NewErrDatabase(op, fmt.Sprintf("ошибка запроса: %s", err))
	}

	r.Logger.DebugContext(ctx, "удаление пользователя",
		"операция", op,
		"id пользователя", id,
	)

	result, err := r.Client.Exec(ctx, sql, args...)
	if err != nil {
		return 0, r.HandleError(op, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		r.Logger.DebugContext(ctx, "пользователь ненайке",
			"операция", op,
			"id пользователя", id,
		)
		return 0, basedberrors.NewErrNotFound("user", id)
	}

	r.Logger.InfoContext(ctx, "пользователь успешно удален",
		"операция", op,
		"id пользователя", id,
		"строк удалено", rowsAffected,
	)

	return id, nil
}

func (r *Repository) applyFilters(query squirrel.SelectBuilder, filter user.Filter) squirrel.SelectBuilder {
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
