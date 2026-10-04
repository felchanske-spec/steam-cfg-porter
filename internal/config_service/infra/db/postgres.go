package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
)

//go:embed../migrations/*.sql
var embedMigrations embed.FS

func InitPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	gooseDb, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open temporary connection for goose: %w", err)
	}
	defer gooseDb.Close()

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	log.Println("[Goose] Running database migrations...")
	if err := goose.Up(gooseDb, "../migrations"); err != nil {
		return nil, fmt.Errorf("goose up failed: %w", err)
	}
	log.Println("[Goose] Migrations applied successfully!")

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dsn configuration: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping native pgx pool: %w", err)
	}

	log.Println("[Postgres] Native pgxpool v5 successfully initialized")
	return pool, nil
}
