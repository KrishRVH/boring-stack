package server

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/KrishRVH/boring-stack/internal/db"
	"github.com/KrishRVH/boring-stack/internal/db/migrations"
	"github.com/KrishRVH/boring-stack/internal/realtime"
)

func TestHTTPHTMXTodoMutations(t *testing.T) {
	app, cleanup := newIntegrationApp(t)
	defer cleanup()

	body := "http integration " + strings.ReplaceAll(t.Name(), "/", "-")
	create := serveHTMX(t, app, http.MethodPost, "/todos", url.Values{"body": {body}}.Encode())
	assertStatus(t, create, http.StatusOK)
	assertContains(t, create.Body.String(), `id="composer-panel"`)
	assertContains(t, create.Body.String(), `hx-swap-oob="outerHTML"`)

	todos, err := app.q.ListTodos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var id string
	for _, todo := range todos {
		if todo.Body == body {
			id = todo.ID
			break
		}
	}
	if id == "" {
		t.Fatalf("created todo %q not found", body)
	}

	toggle := serveHTMX(t, app, http.MethodPost, "/todos/"+id+"/toggle", "")
	assertStatus(t, toggle, http.StatusOK)
	assertContains(t, toggle.Body.String(), `id="todos-panel"`)
	assertContains(t, toggle.Body.String(), `hx-swap-oob="outerHTML"`)

	events, err := app.q.ListEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	assertEvent(t, events, "created: "+body)
	assertEvent(t, events, "marked done: "+body)

	deleteResp := serveHTMX(t, app, http.MethodDelete, "/todos/"+id, "")
	assertStatus(t, deleteResp, http.StatusOK)
	assertContains(t, deleteResp.Body.String(), `id="events-panel"`)

	todos, err = app.q.ListTodos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, todo := range todos {
		if todo.ID == id {
			t.Fatalf("deleted todo %s still exists", id)
		}
	}
	events, err = app.q.ListEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	assertEvent(t, events, "deleted: "+body)
}

func TestStreamSendsInitialSnapshot(t *testing.T) {
	app, cleanup := newIntegrationApp(t)
	defer cleanup()

	ts := httptest.NewServer(app.mux)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("Content-Type"); got != "text/event-stream; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	body := readUntil(t, resp.Body, `hx-swap-oob="outerHTML"`)
	assertContains(t, body, "event: snapshot")
	assertContains(t, body, `id="stats-panel"`)
	assertContains(t, body, `id="todos-panel"`)
	assertContains(t, body, `id="events-panel"`)
	assertContains(t, body, `hx-swap-oob="outerHTML"`)
}

func TestShutdownClosesOpenStream(t *testing.T) {
	app, cleanup := newIntegrationApp(t)
	defer cleanup()

	listenCtx, listenCancel := context.WithTimeout(context.Background(), time.Second)
	defer listenCancel()
	ln, err := (&net.ListenConfig{}).Listen(listenCtx, "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() { errCh <- app.server.Serve(ln) }()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+ln.Addr().String()+"/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	readUntil(t, resp.Body, `hx-swap-oob="outerHTML"`)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	if err := app.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve: %v", err)
	}
}

func TestSnapshotJobRequiresRiver(t *testing.T) {
	app, cleanup := newIntegrationApp(t)
	defer cleanup()

	resp := serveHTMX(t, app, http.MethodPost, "/jobs/snapshot", "")
	assertStatus(t, resp, http.StatusServiceUnavailable)
}

func TestShowcaseActionsReturnSnapshots(t *testing.T) {
	app, cleanup := newIntegrationApp(t)
	defer cleanup()

	pulse := serveHTMX(t, app, http.MethodPost, "/demo/pulse", "")
	assertStatus(t, pulse, http.StatusOK)
	assertContains(t, pulse.Body.String(), `id="composer-panel"`)
	events, err := app.q.ListEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	assertEvent(t, events, "manual pulse over memory bus")

	seed := serveHTMX(t, app, http.MethodPost, "/demo/seed", "")
	assertStatus(t, seed, http.StatusOK)
	assertContains(t, seed.Body.String(), `id="composer-panel"`)
	todos, err := app.q.ListTodos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := countTodoBody(todos, "Model the account billing workflow"); got != 1 {
		t.Fatalf("seeded todo count = %d, want 1", got)
	}

	reseed := serveHTMX(t, app, http.MethodPost, "/demo/seed", "")
	assertStatus(t, reseed, http.StatusOK)
	todos, err = app.q.ListTodos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := countTodoBody(todos, "Model the account billing workflow"); got != 1 {
		t.Fatalf("seeded todo count after reseed = %d, want 1", got)
	}
	events, err = app.q.ListEvents(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	assertEvent(t, events, "seeded 3 app-building steps")
	assertEvent(t, events, "showcase seed already present")
}

func newIntegrationApp(t *testing.T) (*App, func()) {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("set TEST_DATABASE_URL to run server integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	migrateServerTestDB(ctx, t, dbURL)
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	resetServerTestData(ctx, t, pool)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := New(":0", logger, pool, realtime.NewMemoryBus(), nil)
	cleanup := func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Exec(cleanupCtx, `TRUNCATE todos, app_events RESTART IDENTITY CASCADE`)
		pool.Close()
	}
	return app, cleanup
}

func migrateServerTestDB(ctx context.Context, t *testing.T, dbURL string) {
	t.Helper()

	sqlDB, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sqlDB.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
}

func resetServerTestData(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE todos, app_events RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
}

func serveHTMX(t *testing.T, app *App, method string, target string, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), method, target, strings.NewReader(body))
	req.Header.Set("HX-Request", "true")
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()
	app.mux.ServeHTTP(rec, req)
	return rec
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, want, rec.Body.String())
	}
}

func assertContains(t *testing.T, got string, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("response missing %q\n%s", want, got)
	}
}

func assertEvent(t *testing.T, events []db.AppEvent, want string) {
	t.Helper()
	for _, event := range events {
		if event.Body == want {
			return
		}
	}
	t.Fatalf("event %q not found in %#v", want, events)
}

func countTodoBody(todos []db.Todo, body string) int {
	count := 0
	for _, todo := range todos {
		if todo.Body == body {
			count++
		}
	}
	return count
}

func readUntil(t *testing.T, r io.Reader, want string) string {
	t.Helper()

	var b strings.Builder
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read stream: %v", err)
		}
		b.WriteString(line)
		body := b.String()
		if strings.Contains(body, want) {
			return body
		}
	}
}
