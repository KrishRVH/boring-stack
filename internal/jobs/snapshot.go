package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/KrishRVH/boring-stack/internal/db"
	"github.com/KrishRVH/boring-stack/internal/realtime"
)

// SnapshotArgs contains the reason a snapshot job was requested.
type SnapshotArgs struct {
	Reason string `json:"reason"`
}

// Kind identifies snapshot jobs to River.
func (SnapshotArgs) Kind() string { return "snapshot" }

// SnapshotWorker records a todo snapshot when River runs a snapshot job.
type SnapshotWorker struct {
	river.WorkerDefaults[SnapshotArgs]

	Queries *db.Queries
	Bus     realtime.Bus
	Logger  *slog.Logger
}

// NewSnapshotWorker builds a SnapshotWorker.
func NewSnapshotWorker(queries *db.Queries, bus realtime.Bus, logger *slog.Logger) *SnapshotWorker {
	return &SnapshotWorker{Queries: queries, Bus: bus, Logger: logger}
}

// Work records the current todo counts and publishes the change.
func (w *SnapshotWorker) Work(ctx context.Context, job *river.Job[SnapshotArgs]) error {
	stats, err := w.Queries.CountTodos(ctx)
	if err != nil {
		return err
	}

	body := fmt.Sprintf(
		"River job %d processed at %s: %d total, %d open, %d done. Reason: %s",
		job.ID,
		time.Now().Format(time.RFC3339),
		stats.Total,
		stats.Total-stats.Done,
		stats.Done,
		job.Args.Reason,
	)

	if _, err := w.Queries.InsertEvent(ctx, db.InsertEventParams{Kind: "river", Body: body}); err != nil {
		return err
	}
	if err := w.Bus.Publish(context.WithoutCancel(ctx), realtime.TopicTodosChanged, []byte("river")); err != nil {
		w.Logger.Warn("bus publish failed after River event insert", "job_id", job.ID, "err", err)
	}
	w.Logger.Info("processed River snapshot job", "job_id", job.ID, "reason", job.Args.Reason)
	return nil
}
