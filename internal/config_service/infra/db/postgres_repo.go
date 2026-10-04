package db

import (
	"context"
	"errors"
	"fmt"
	"steam-cfg-portet/internal/config_service/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) domain.PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Get(ctx context.Context, id string, game string, authorNickname string) (*domain.Config, error) {
	query := `
		SELECT id, game, name, author_nickname, is_global, user_id, config_body, created_at 
		FROM configs 
		WHERE id = $1 AND game = $2 AND author_nickname = $3
	`

	cfg := &domain.Config{}

	err := r.pool.QueryRow(ctx, query, id, game, authorNickname).Scan(
		&cfg.ID,
		&cfg.Game,
		&cfg.Name,
		&cfg.AuthorNickName,
		&cfg.IsGlobal,
		&cfg.UserID,
		&cfg.ConfigBody,
		&cfg.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrConfigNotFound
		}
		return nil, fmt.Errorf("postgres Get failed: %v", err)
	}

	return cfg, nil
}

func (r *PostgresRepo) Save(ctx context.Context, cfg *domain.Config) (string, error) {
	query := `
		INSERT INTO configs (id, game, name, author_nickname, is_global, user_id, config_body, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	var returnedID string

	err := r.pool.QueryRow(ctx, query,
		cfg.ID,
		cfg.Game,
		cfg.Name,
		cfg.AuthorNickName,
		cfg.IsGlobal,
		cfg.UserID,
		cfg.ConfigBody,
		cfg.CreatedAt,
	).Scan(&returnedID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 — код ошибки unique_violation в Postgres
			return "", domain.ErrConfigAlreadyExists
		}
		return "", fmt.Errorf("postgres Save failed: %v", err)
	}

	return returnedID, nil
}

func (r *PostgresRepo) Delete(ctx context.Context, id string, userID int64) (bool, error) {
	query := `
		DELETE FROM configs 
		WHERE id = $1 AND user_id = $2
	`

	result, err := r.pool.Exec(ctx, query, id, userID)
	if err != nil {
		return false, fmt.Errorf("postgres Delete failed: %v", err)
	}

	return result.RowsAffected() > 0, nil
}
