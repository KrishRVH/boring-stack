package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/KrishRVH/boring-stack/internal/db"
)

func TestMigrationsAndQueries(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("set TEST_DATABASE_URL to run migration/query integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	if err := migrateGoose(ctx, dbURL, logger); err != nil {
		t.Fatalf("goose migrations: %v", err)
	}
	if err := migrateRiver(ctx, dbURL, logger); err != nil {
		t.Fatalf("river migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	resetTestData(t, ctx, pool)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `TRUNCATE todos, app_events RESTART IDENTITY CASCADE`)
	}()

	var hasRiverJob bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.river_job') IS NOT NULL`).Scan(&hasRiverJob); err != nil {
		t.Fatal(err)
	}
	if !hasRiverJob {
		t.Fatal("river_job table does not exist")
	}

	queries := db.New(pool)
	body := "integration todo " + strings.ReplaceAll(t.Name(), "/", "-")
	created, err := queries.CreateTodo(ctx, body)
	if err != nil {
		t.Fatalf("CreateTodo: %v", err)
	}

	toggled, err := queries.ToggleTodo(ctx, created.ID)
	if err != nil {
		t.Fatalf("ToggleTodo: %v", err)
	}
	if !toggled.Done {
		t.Fatal("ToggleTodo did not mark todo done")
	}

	if _, err := queries.InsertEvent(ctx, db.InsertEventParams{Kind: "test", Body: body}); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	deleted, err := queries.DeleteTodo(ctx, created.ID)
	if err != nil {
		t.Fatalf("DeleteTodo: %v", err)
	}
	if deleted.ID != created.ID {
		t.Fatalf("DeleteTodo deleted %q, want %q", deleted.ID, created.ID)
	}
}

func resetTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE todos, app_events RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
}
