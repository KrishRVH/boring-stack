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
	resetTestData(ctx, t, pool)
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

	assertTodoConstraints(ctx, t, pool)
	queries := db.New(pool)
	assertTodoBodyConstraints(ctx, t, queries)

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

func assertTodoConstraints(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	var count int
	var hasBodyCheck, hasMaxLengthCheck bool
	if err := pool.QueryRow(ctx, `
SELECT
	count(*)::int,
	bool_or(conname = 'todos_body_check'),
	bool_or(conname = 'todos_body_max_length_check')
FROM pg_constraint
WHERE conrelid = 'todos'::regclass
	AND contype = 'c'
`).Scan(&count, &hasBodyCheck, &hasMaxLengthCheck); err != nil {
		t.Fatal(err)
	}
	if count != 2 || !hasBodyCheck || !hasMaxLengthCheck {
		t.Fatalf("todo check constraints = count %d, body %t, max length %t", count, hasBodyCheck, hasMaxLengthCheck)
	}
}

func assertTodoBodyConstraints(ctx context.Context, t *testing.T, queries *db.Queries) {
	t.Helper()

	if _, err := queries.CreateTodo(ctx, "   "); err == nil {
		t.Fatal("CreateTodo accepted a blank body")
	}
	if _, err := queries.CreateTodo(ctx, strings.Repeat("界", 281)); err == nil {
		t.Fatal("CreateTodo accepted a 281-character body")
	}
	if _, err := queries.CreateTodo(ctx, strings.Repeat("界", 280)); err != nil {
		t.Fatalf("CreateTodo rejected a 280-character body: %v", err)
	}
}

func resetTestData(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE todos, app_events RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
}
