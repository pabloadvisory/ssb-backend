package maintenance

import (
	"context"
	"log/slog"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/observability"
)

type Result struct {
	Deliveries int64
	Outbox     int64
	Audit      int64
}

type Store interface {
	Prune(context.Context, time.Time, time.Time, time.Time, int) (Result, error)
}

type Config struct {
	Interval          time.Duration
	DeliveryRetention time.Duration
	OutboxRetention   time.Duration
	AuditRetention    time.Duration
	BatchSize         int
}

type Worker struct {
	store   Store
	logger  *slog.Logger
	config  Config
	metrics observability.Metrics
}

func NewWorker(store Store, logger *slog.Logger, metrics observability.Metrics, config Config) *Worker {
	if metrics == nil {
		metrics = observability.NopMetrics{}
	}
	return &Worker{store: store, logger: logger, config: config, metrics: metrics}
}

func (worker *Worker) Run(ctx context.Context) {
	worker.prune(ctx)
	ticker := time.NewTicker(worker.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			worker.prune(ctx)
		}
	}
}

func (worker *Worker) prune(ctx context.Context) {
	now := time.Now()
	result, err := worker.store.Prune(
		ctx,
		now.Add(-worker.config.DeliveryRetention),
		now.Add(-worker.config.OutboxRetention),
		now.Add(-worker.config.AuditRetention),
		worker.config.BatchSize,
	)
	if err != nil {
		if ctx.Err() == nil {
			worker.logger.Error("prune retained records", "error", err)
			worker.metrics.ObserveWorkerResult("maintenance", "failed")
		}
		return
	}
	worker.metrics.ObserveWorkerResult("maintenance", "completed")
	if result.Deliveries+result.Outbox+result.Audit > 0 {
		worker.logger.Info("retained records pruned",
			"notification_deliveries", result.Deliveries,
			"outbox_events", result.Outbox,
			"endpoint_audit", result.Audit,
		)
	}
}
