package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/KrishRVH/boring-stack/internal/config"
	"github.com/KrishRVH/boring-stack/internal/db/migrations"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

	if err := migrateGoose(ctx, cfg.DBURL, logger); err != nil {
		logger.Error("goose migration failed", "err", err)
		cancel()
		os.Exit(1)
	}
	if err := migrateRiver(ctx, cfg.DBURL, logger); err != nil {
		logger.Error("river migration failed", "err", err)
		cancel()
		os.Exit(1)
	}
	cancel()
}

func migrateGoose(ctx context.Context, dbURL string, logger *slog.Logger) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS, goose.WithSlog(logger))
	if err != nil {
		return err
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return err
	}
	for _, result := range results {
		logger.Info("goose migration applied", "version", result.Source.Version, "duration", result.Duration)
	}
	return nil
}

func migrateRiver(ctx context.Context, dbURL string, logger *slog.Logger) error {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), &rivermigrate.Config{Logger: logger})
	if err != nil {
		return err
	}
	result, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return err
	}
	for _, version := range result.Versions {
		logger.Info("river migration applied", "version", version.Version, "name", version.Name, "duration", version.Duration)
	}
	return nil
}
