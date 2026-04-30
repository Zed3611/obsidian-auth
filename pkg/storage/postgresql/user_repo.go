package postgresqlstorage

import (
	"context"
	"errors"
	"fmt"
	"obsidian-auth/pkg/domain/models"
	"obsidian-auth/pkg/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	pg "github.com/jackc/pgx/v5/pgxpool"

	"github.com/jackc/pgerrcode"
)

type PgUserStorage struct {
	pool *pg.Pool
}

func New(pool *pg.Pool) *PgUserStorage {
	return &PgUserStorage{pool}
}

func (s *PgUserStorage) Create(ctx context.Context, email, passwordHash string) (*models.User, error) {
	const op = "postgresql.user_repo.Create"

	const query = `--sql
		insert into users(email, password_hash) 
		values ($1, $2)
		returning id, email, password_hash, created_at, updated_at
	`

	var user models.User

	err := s.pool.QueryRow(ctx, query, email, passwordHash).Scan(
		&user.Id,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

func (s *PgUserStorage) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	const op = "postgresql.user_repo.FindByEmail"

	const query = `--sql
		select * from users where email = '$1'
	`

	var user models.User

	err := s.pool.QueryRow(ctx, query, email).Scan(
		&user.Id,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}

func (s *PgUserStorage) FindByID(ctx context.Context, id int) (*models.User, error) {
	const op = "postgresql.user_repo.FindByEmail"

	const query = `--sql
		select * from users where id = $1
	`

	var user models.User

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&user.Id,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrUserNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &user, nil
}
