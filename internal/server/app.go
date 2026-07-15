package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/KrishRVH/boring-stack/internal/appmodel"
	"github.com/KrishRVH/boring-stack/internal/config"
	"github.com/KrishRVH/boring-stack/internal/db"
	"github.com/KrishRVH/boring-stack/internal/jobs"
	"github.com/KrishRVH/boring-stack/internal/realtime"
	"github.com/KrishRVH/boring-stack/internal/ui"
	webassets "github.com/KrishRVH/boring-stack/web"
)

// App serves the HTTP application.
type App struct {
	cfg    config.Config
	log    *slog.Logger
	pool   *pgxpool.Pool
	q      *db.Queries
	bus    realtime.Bus
	river  *river.Client[pgx.Tx]
	mux    *http.ServeMux
	server *http.Server

	streamShutdown context.Context
	stopStreams    context.CancelFunc
}

const (
	maxFormBodyBytes = 16 << 10
	todoEventKind    = "todo"
	todoBodyRequired = "Todo body is required."
)

// New builds an App and registers its routes.
func New(cfg config.Config, logger *slog.Logger, pool *pgxpool.Pool, bus realtime.Bus, riverClient *river.Client[pgx.Tx]) *App {
	streamShutdown, stopStreams := context.WithCancel(context.Background())
	app := &App{
		cfg:            cfg,
		log:            logger,
		pool:           pool,
		q:              db.New(pool),
		bus:            bus,
		river:          riverClient,
		mux:            http.NewServeMux(),
		streamShutdown: streamShutdown,
		stopStreams:    stopStreams,
	}
	app.routes()
	app.server = &http.Server{
		Addr:              cfg.Addr,
		Handler:           app.requestLogger(app.securityHeaders(app.mux)),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	return app
}

// ListenAndServe starts the HTTP server.
func (a *App) ListenAndServe() error {
	a.log.Info("server listening", "addr", a.cfg.Addr, "bus", a.bus.Name())
	if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown stops open streams and gracefully shuts down the HTTP server.
func (a *App) Shutdown(ctx context.Context) error {
	a.stopStreams()
	return a.server.Shutdown(ctx)
}

func (a *App) routes() {
	a.mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(webassets.Assets())))
	a.mux.HandleFunc("GET /healthz", a.healthz)
	a.mux.HandleFunc("GET /readyz", a.readyz)
	a.mux.HandleFunc("GET /{$}", a.home)
	a.mux.HandleFunc("GET /stream", a.stream)
	a.mux.HandleFunc("POST /todos", a.createTodo)
	a.mux.HandleFunc("POST /todos/{id}/toggle", a.toggleTodo)
	a.mux.HandleFunc("DELETE /todos/{id}", a.deleteTodo)
	a.mux.HandleFunc("POST /jobs/snapshot", a.enqueueSnapshot)
	a.mux.HandleFunc("POST /demo/pulse", a.broadcastPulse)
	a.mux.HandleFunc("POST /demo/seed", a.seedShowcase)
}

func (a *App) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) readyz(w http.ResponseWriter, r *http.Request) {
	var appSchemaReady, riverSchemaReady bool
	err := a.pool.QueryRow(r.Context(), `
SELECT
	to_regclass('public.todos') IS NOT NULL
		AND to_regclass('public.app_events') IS NOT NULL,
	to_regclass('public.river_job') IS NOT NULL
`).Scan(&appSchemaReady, &riverSchemaReady)
	if err != nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	if !appSchemaReady || !riverSchemaReady {
		http.Error(w, "database schema not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	vm, err := a.viewModel(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	vm.FormError = r.URL.Query().Get("todo_error")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.Home(vm).Render(r.Context(), w); err != nil {
		a.serverError(w, r, err)
	}
}

func (a *App) stream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	events, unsubscribe, err := a.bus.Subscribe(ctx, realtime.TopicTodosChanged)
	if err != nil {
		a.log.Error("bus subscribe failed", "err", err)
		http.Error(w, "stream unavailable", http.StatusServiceUnavailable)
		return
	}
	defer unsubscribe()

	var initial bytes.Buffer
	if err := a.writeSnapshotEvent(ctx, &initial); err != nil {
		a.log.Error("initial stream patch failed", "err", err)
		a.serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if _, err := w.Write(initial.Bytes()); err != nil {
		return
	}
	flusher.Flush()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	a.runStream(ctx, w, flusher, events, heartbeat.C)
}

func (a *App) runStream(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, events <-chan realtime.Event, heartbeat <-chan time.Time) {
	for {
		select {
		case <-a.streamShutdown.Done():
			return
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			a.log.Debug("stream patch", "topic", event.Topic, "data", string(event.Data))
			if err := a.writeSnapshotEvent(ctx, w); err != nil {
				a.log.Error("stream patch failed", "err", err)
				return
			}
			flusher.Flush()
		case <-heartbeat:
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (a *App) createTodo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBodyBytes)
	if err := r.ParseForm(); err != nil {
		a.badRequest(w, r, fmt.Errorf("read form: %w", err))
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if formError := validateTodoBody(body); formError != "" {
		a.formError(w, r, formError)
		return
	}

	var created db.Todo
	err := a.withTx(r.Context(), func(qtx *db.Queries) error {
		var err error
		created, err = qtx.CreateTodo(r.Context(), body)
		if err != nil {
			return err
		}
		_, err = qtx.InsertEvent(r.Context(), db.InsertEventParams{Kind: todoEventKind, Body: "created: " + created.Body})
		return err
	})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.publishChange(r.Context(), "create")

	if !isHTMX(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := a.writeCreateResponse(r.Context(), w); err != nil {
		a.serverError(w, r, err)
	}
}

func (a *App) toggleTodo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var updated db.Todo
	err := a.withTx(r.Context(), func(qtx *db.Queries) error {
		var err error
		updated, err = qtx.ToggleTodo(r.Context(), id)
		if err != nil {
			return err
		}
		state := "open"
		if updated.Done {
			state = "done"
		}
		body := fmt.Sprintf("marked %s: %s", state, updated.Body)
		_, err = qtx.InsertEvent(r.Context(), db.InsertEventParams{Kind: todoEventKind, Body: body})
		return err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			a.notFoundOrSnapshot(w, r)
			return
		}
		a.serverError(w, r, err)
		return
	}
	a.publishChange(r.Context(), "toggle")

	a.writeSnapshotResponse(w, r)
}

func (a *App) deleteTodo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var deleted db.Todo
	err := a.withTx(r.Context(), func(qtx *db.Queries) error {
		var err error
		deleted, err = qtx.DeleteTodo(r.Context(), id)
		if err != nil {
			return err
		}
		_, err = qtx.InsertEvent(r.Context(), db.InsertEventParams{Kind: todoEventKind, Body: "deleted: " + deleted.Body})
		return err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			a.notFoundOrSnapshot(w, r)
			return
		}
		a.serverError(w, r, err)
		return
	}
	a.publishChange(r.Context(), "delete")

	a.writeSnapshotResponse(w, r)
}

func (a *App) enqueueSnapshot(w http.ResponseWriter, r *http.Request) {
	if a.river == nil {
		http.Error(w, "snapshot jobs are not available", http.StatusServiceUnavailable)
		return
	}

	tx, err := a.pool.Begin(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(r.Context())) }()

	qtx := a.q.WithTx(tx)

	inserted, err := a.river.InsertTx(r.Context(), tx, jobs.SnapshotArgs{Reason: "manual button click"}, nil)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	body := fmt.Sprintf("enqueued snapshot job %d", inserted.Job.ID)
	if _, err := qtx.InsertEvent(r.Context(), db.InsertEventParams{Kind: "river", Body: body}); err != nil {
		a.serverError(w, r, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		a.serverError(w, r, err)
		return
	}
	a.publishChange(r.Context(), "job-enqueued")

	a.writeComposerClearResponse(w, r)
}

func (a *App) broadcastPulse(w http.ResponseWriter, r *http.Request) {
	body := fmt.Sprintf("manual pulse over %s bus", a.bus.Name())
	if _, err := a.q.InsertEvent(r.Context(), db.InsertEventParams{Kind: "pulse", Body: body}); err != nil {
		a.serverError(w, r, err)
		return
	}
	a.publishChange(r.Context(), "pulse")

	a.writeComposerClearResponse(w, r)
}

func (a *App) seedShowcase(w http.ResponseWriter, r *http.Request) {
	missing := showcaseSeedItems

	err := a.withTx(r.Context(), func(qtx *db.Queries) error {
		todos, err := qtx.ListTodos(r.Context())
		if err != nil {
			return err
		}
		missing = missingShowcaseSeedItems(todos)

		for _, item := range missing {
			if _, err := qtx.CreateTodo(r.Context(), item); err != nil {
				return err
			}
		}

		body := "showcase seed already present"
		if len(missing) > 0 {
			body = fmt.Sprintf("seeded %d app-building steps", len(missing))
		}
		_, err = qtx.InsertEvent(r.Context(), db.InsertEventParams{Kind: "seed", Body: body})
		return err
	})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.publishChange(r.Context(), "seed")

	a.writeComposerClearResponse(w, r)
}

var showcaseSeedItems = []string{
	"Model the account billing workflow",
	"Render the operator review queue",
	"Move invoice sync into River",
}

func missingShowcaseSeedItems(todos []db.Todo) []string {
	existing := make(map[string]bool, len(todos))
	for _, todo := range todos {
		existing[todo.Body] = true
	}

	missing := make([]string, 0, len(showcaseSeedItems))
	for _, item := range showcaseSeedItems {
		if !existing[item] {
			missing = append(missing, item)
		}
	}
	return missing
}

func (a *App) publishChange(ctx context.Context, reason string) {
	if err := a.bus.Publish(context.WithoutCancel(ctx), realtime.TopicTodosChanged, []byte(reason)); err != nil {
		a.log.Warn("bus publish failed", "topic", realtime.TopicTodosChanged, "reason", reason, "err", err)
	}
}

func (a *App) withTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	if err := fn(a.q.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (a *App) writeCreateResponse(ctx context.Context, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.TodoComposer("").Render(ctx, w); err != nil {
		return err
	}
	return a.renderSnapshot(ctx, w)
}

func (a *App) writeComposerResponse(ctx context.Context, w http.ResponseWriter, formError string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return ui.TodoComposer(formError).Render(ctx, w)
}

func (a *App) writeComposerClearResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.TodoComposer("").Render(r.Context(), w); err != nil {
		a.serverError(w, r, err)
	}
}

func (a *App) writeSnapshotResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.renderSnapshot(r.Context(), w); err != nil {
		a.serverError(w, r, err)
	}
}

func (a *App) renderSnapshot(ctx context.Context, w io.Writer) error {
	vm, err := a.viewModel(ctx)
	if err != nil {
		return err
	}
	if err := ui.StatsPanel(vm.Stats, true).Render(ctx, w); err != nil {
		return err
	}
	if err := ui.TodosPanel(vm.Todos, true).Render(ctx, w); err != nil {
		return err
	}
	return ui.EventsPanel(vm.Events, true).Render(ctx, w)
}

func (a *App) writeSnapshotEvent(ctx context.Context, w io.Writer) error {
	var b bytes.Buffer
	if err := a.renderSnapshot(ctx, &b); err != nil {
		return err
	}
	return writeSSE(w, "snapshot", b.String())
}

func writeSSE(w io.Writer, event string, data string) error {
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	normalized := strings.ReplaceAll(data, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	for _, line := range strings.Split(normalized, "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}

func validateTodoBody(body string) string {
	body = strings.TrimSpace(body)
	switch {
	case body == "":
		return todoBodyRequired
	case utf8.RuneCountInString(body) > appmodel.MaxTodoBodyLength:
		return fmt.Sprintf("Todo body must be %d characters or fewer.", appmodel.MaxTodoBodyLength)
	default:
		return ""
	}
}

func (a *App) formError(w http.ResponseWriter, r *http.Request, message string) {
	if !isHTMX(r) {
		http.Redirect(w, r, "/?todo_error="+url.QueryEscape(message), http.StatusSeeOther)
		return
	}
	if err := a.writeComposerResponse(r.Context(), w, message); err != nil {
		a.serverError(w, r, err)
	}
}

func (a *App) notFoundOrSnapshot(w http.ResponseWriter, r *http.Request) {
	if !isHTMX(r) {
		http.NotFound(w, r)
		return
	}
	a.writeSnapshotResponse(w, r)
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

func (a *App) viewModel(ctx context.Context) (appmodel.HomeView, error) {
	todos, err := a.q.ListTodos(ctx)
	if err != nil {
		return appmodel.HomeView{}, err
	}
	stats, err := a.q.CountTodos(ctx)
	if err != nil {
		return appmodel.HomeView{}, err
	}
	events, err := a.q.ListEvents(ctx, 10)
	if err != nil {
		return appmodel.HomeView{}, err
	}

	vm := appmodel.HomeView{
		Todos:   make([]appmodel.Todo, 0, len(todos)),
		Events:  make([]appmodel.Event, 0, len(events)),
		Stats:   appmodel.Stats{Total: stats.Total, Done: stats.Done},
		BusName: a.bus.Name(),
		Version: versionInfo(),
	}
	for _, todo := range todos {
		vm.Todos = append(vm.Todos, appmodel.Todo{
			ID:        todo.ID,
			Body:      todo.Body,
			Done:      todo.Done,
			CreatedAt: todo.CreatedAt,
			UpdatedAt: todo.UpdatedAt,
		})
	}
	for _, event := range events {
		vm.Events = append(vm.Events, appmodel.Event{
			ID:        event.ID,
			Kind:      event.Kind,
			Body:      event.Body,
			CreatedAt: event.CreatedAt,
		})
	}
	return vm, nil
}

func versionInfo() appmodel.VersionInfo {
	return appmodel.VersionInfo{
		Go:       runtime.Version(),
		HTMX:     envVersion("HTMX_VERSION"),
		HTMXSSE:  envVersion("HTMX_SSE_VERSION"),
		Alpine:   envVersion("ALPINE_VERSION"),
		Templ:    moduleVersion("github.com/a-h/templ", envVersion("TEMPL_VERSION")),
		Tailwind: envVersion("TAILWIND_VERSION"),
		SQLC:     envVersion("SQLC_VERSION"),
		PGX:      moduleVersion("github.com/jackc/pgx/v5", "unknown"),
		Goose:    moduleVersion("github.com/pressly/goose/v3", envVersion("GOOSE_VERSION")),
		River:    moduleVersion("github.com/riverqueue/river", "unknown"),
		NATSGo:   moduleVersion("github.com/nats-io/nats.go", "unknown"),
	}
}

func envVersion(key string) string {
	value := os.Getenv(key)
	if value == "" {
		return "not set"
	}
	return value
}

func moduleVersion(path, fallback string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return fallback
	}
	for _, dep := range info.Deps {
		if dep.Path == path {
			if dep.Replace != nil {
				return dep.Replace.Version
			}
			return dep.Version
		}
	}
	return fallback
}

func (a *App) badRequest(w http.ResponseWriter, r *http.Request, err error) {
	a.log.Warn("bad request", "method", r.Method, "path", r.URL.Path, "err", err)
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func (a *App) serverError(w http.ResponseWriter, r *http.Request, err error) {
	a.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (a *App) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		a.log.Debug("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}
