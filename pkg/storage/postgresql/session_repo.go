package postgresqlstorage

import (
	"context"
	"errors"
	"fmt"
	"obsidian-auth/pkg/domain/models"
	"obsidian-auth/pkg/storage"
	"time"

	"github.com/jackc/pgx/v5"
	pg "github.com/jackc/pgx/v5/pgxpool"
)

type PgSessionStorage struct {
	pool *pg.Pool
}

func NewSessionStorage(pool *pg.Pool) *PgSessionStorage {
	return &PgSessionStorage{pool: pool}
}

func (s *PgSessionStorage) Create(ctx context.Context, userId int, ip, userAgent, tokenHash string, activeTil time.Time) (*models.Session, error) {
	const op = "postgresql.session_repo.Create"

	const query = `
		insert into sessions(user_id, ip, user_agent, token_hash, active_til)
		values ($1, $2, $3, $4, $5)
		returning id, user_id, ip, user_agent, token_hash, active_til, refresh_count, created_at, updated_at
	`

	var session models.Session

	err := s.pool.QueryRow(ctx, query, userId, ip, userAgent, tokenHash, activeTil).Scan(
		&session.Id,
		&session.UserId,
		&session.Ip,
		&session.UserAgent,
		&session.RefreshTokenHash,
		&session.ActiveTil,
		&session.RefreshCount,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &session, nil
}

func (s *PgSessionStorage) FindByRefreshTokenHash(ctx context.Context, hash string) (*models.Session, error) {
	const op = "postgresql.session_repo.FindByRefreshTokenHash"

	const query = `
		select id, user_id, ip, user_agent, token_hash, active_til, refresh_count, created_at, updated_at from sessions where token_hash = $1
	`

	var session models.Session

	err := s.pool.QueryRow(ctx, query, hash).Scan(
		&session.Id,
		&session.UserId,
		&session.Ip,
		&session.UserAgent,
		&session.RefreshTokenHash,
		&session.ActiveTil,
		&session.RefreshCount,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrSessionNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &session, nil
}

func (s *PgSessionStorage) UpdateRefreshToken(ctx context.Context, sessionId int, newHash string, activeTil time.Time) error {
	const op = "postgresql.session_repo.UpdateRefreshToken"

	const query = `
		update sessions set (token_hash = $1) where id = $2
	`

	tag, err := s.pool.Exec(ctx, query, newHash, sessionId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrSessionNotFound)
	}

	return nil
}

func (s *PgSessionStorage) FindByUserID(ctx context.Context, userId int) (*[]models.Session, error) {
	const op = "postgresql.session_repo.FindByUserID"

	const query = `
		select id, user_id, ip, user_agent, token_hash, active_til, refresh_count, created_at, updated_at from sessions where user_id = $1
	`

	rows, err := s.pool.Query(ctx, query, userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, storage.ErrSessionNotFound)
		}

		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	var sessions []models.Session

	for rows.Next() {
		var session models.Session

		err := rows.Scan(
			&session.Id,
			&session.UserId,
			&session.Ip,
			&session.UserAgent,
			&session.RefreshTokenHash,
			&session.ActiveTil,
			&session.RefreshCount,
			&session.CreatedAt,
			&session.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		sessions = append(sessions, session)
	}

	return &sessions, nil
}

func (s *PgSessionStorage) DeleteOne(ctx context.Context, sessionId, userId int) error {
	const op = "postgresql.session_repo.DeleteOne"

	const query = `--sql
		delete from sessions where id = $1 and user_id = $2
	`

	tag, err := s.pool.Exec(ctx, query, sessionId, userId)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrSessionNotFound)
	}

	return nil
}

func (s *PgSessionStorage) DeleteManyExcept(ctx context.Context, userId, exceptSessionId int) ([]int, error) {
	const op = "postgresql.session_repo.DeleteOne"

	const query = `--sql
		delete from sessions where user_id = $1 and id != $2
		returning id
	`

	rows, err := s.pool.Query(ctx, query, userId, exceptSessionId)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	defer rows.Close()

	removedIds, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return removedIds, nil
}
