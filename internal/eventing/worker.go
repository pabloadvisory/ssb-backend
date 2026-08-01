package eventing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/observability"
)

type Worker struct {
	store          Store
	logger         *slog.Logger
	concurrency    int
	maxAttempts    int
	pollInterval   time.Duration
	lockDuration   time.Duration
	processTimeout time.Duration
	metrics        observability.Metrics
}

type WorkerConfig struct {
	Concurrency    int
	MaxAttempts    int
	PollInterval   time.Duration
	LockDuration   time.Duration
	ProcessTimeout time.Duration
}

func NewWorker(store Store, logger *slog.Logger, metrics observability.Metrics, config WorkerConfig) *Worker {
	if metrics == nil {
		metrics = observability.NopMetrics{}
	}
	return &Worker{
		store: store, logger: logger, concurrency: config.Concurrency,
		maxAttempts: config.MaxAttempts, pollInterval: config.PollInterval,
		lockDuration: config.LockDuration, processTimeout: config.ProcessTimeout, metrics: metrics,
	}
}

func (worker *Worker) Run(ctx context.Context) {
	for ctx.Err() == nil {
		events, err := worker.store.ClaimOutboxEvents(ctx, worker.concurrency, worker.lockDuration)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			worker.logger.Error("claim outbox events", "error", err)
			if !wait(ctx, worker.pollInterval) {
				return
			}
			continue
		}
		if len(events) == 0 {
			if !wait(ctx, worker.pollInterval) {
				return
			}
			continue
		}
		worker.metrics.ObserveWorkerBatch("outbox", len(events))
		worker.processBatch(ctx, events)
	}
}

func (worker *Worker) processBatch(ctx context.Context, events []Event) {
	var group sync.WaitGroup
	group.Add(len(events))
	for _, event := range events {
		go func() {
			defer group.Done()
			worker.handle(ctx, event)
		}()
	}
	group.Wait()
}

func (worker *Worker) handle(ctx context.Context, event Event) {
	processContext, cancel := context.WithTimeout(ctx, worker.processTimeout)
	defer cancel()

	var err error
	switch event.Type {
	case MatchChangedV1:
		var changed MatchChanged
		if err = json.Unmarshal(event.Payload, &changed); err == nil {
			plans := notification.PlanMatchDeliveries(changed.Notification)
			err = worker.store.PublishMatchChanged(processContext, event, changed, plans)
		}
	default:
		err = fmt.Errorf("unsupported outbox event type %q", event.Type)
	}
	if err == nil {
		worker.metrics.ObserveWorkerResult("outbox", "published")
		return
	}
	if ctx.Err() != nil {
		worker.metrics.ObserveWorkerResult("outbox", "cancelled")
		return
	}
	cancel()
	terminal := event.Attempts >= worker.maxAttempts || event.Type != MatchChangedV1
	backoff := time.Duration(math.Pow(2, float64(min(event.Attempts, 8)))) * time.Second
	if retryErr := worker.store.RetryOutboxEvent(
		ctx, event.ID, event.LockToken, time.Now().Add(backoff), err.Error(), terminal,
	); retryErr != nil {
		worker.logger.Error("reschedule outbox event", "event_id", event.ID, "error", retryErr)
		worker.metrics.ObserveWorkerResult("outbox", "lease_error")
		return
	}
	if terminal {
		worker.metrics.ObserveWorkerResult("outbox", "failed")
	} else {
		worker.metrics.ObserveWorkerResult("outbox", "retry")
	}
	worker.logger.Warn("outbox event failed", "event_id", event.ID, "event_type", event.Type, "attempt", event.Attempts, "terminal", terminal, "error", err)
}

func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
