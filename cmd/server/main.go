package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/KrishRVH/boring-stack/internal/config"
	"github.com/KrishRVH/boring-stack/internal/db"
	"github.com/KrishRVH/boring-stack/internal/jobs"
	"github.com/KrishRVH/boring-stack/internal/realtime"
	"github.com/KrishRVH/boring-stack/internal/server"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DBURL)
	must(logger, err, "connect database")
	defer pool.Close()
	must(logger, pool.Ping(ctx), "ping database")

	bus := buildBus(logger, cfg)
	defer func() {
		if err := bus.Close(); err != nil {
			logger.Error("close bus", "err", err)
		}
	}()

	workers := river.NewWorkers()
	queries := db.New(pool)
	river.AddWorker(workers, jobs.NewSnapshotWorker(queries, bus, logger))

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: logger,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 4},
		},
		Workers: workers,
	})
	must(logger, err, "create River client")
	must(logger, riverClient.Start(ctx), "start River client")
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := riverClient.Stop(shutdownCtx); err != nil {
			logger.Error("stop River client", "err", err)
		}
	}()

	app := server.New(cfg, logger, pool, bus, riverClient)

	errCh := make(chan error, 1)
	go func() { errCh <- app.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		must(logger, app.Shutdown(shutdownCtx), "shutdown server")
	case err := <-errCh:
		must(logger, err, "serve")
	}
}

func buildBus(logger *slog.Logger, cfg config.Config) realtime.Bus {
	switch cfg.Bus {
	case "nats":
		bus, err := realtime.NewNATSBus(cfg.NATSURL)
		must(logger, err, "connect NATS")
		return bus
	case "memory":
		return realtime.NewMemoryBus()
	default:
		must(logger, fmt.Errorf("unknown BUS %q", cfg.Bus), "configure bus")
		return nil
	}
}

func must(logger *slog.Logger, err error, msg string) {
	if err != nil {
		logger.Error(msg, "err", err)
		os.Exit(1)
	}
}
