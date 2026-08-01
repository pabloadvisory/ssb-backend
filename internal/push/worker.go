package push

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/observability"
)

type Worker struct {
	store        DeliveryStore
	dispatcher   Sender
	logger       *slog.Logger
	concurrency  int
	maxAttempts  int
	pollInterval time.Duration
	lockDuration time.Duration
	sendTimeout  time.Duration
	metrics      observability.Metrics
}

type Sender interface {
	Send(context.Context, notification.Delivery) (Result, error)
}

type DeliveryStore interface {
	ClaimDeliveries(context.Context, int, time.Duration) ([]notification.Delivery, error)
	CompleteDelivery(context.Context, string, string, string) error
	RetryDelivery(context.Context, string, string, time.Time, string, bool) error
	InvalidateEndpoint(context.Context, string, string) error
}

type WorkerConfig struct {
	Concurrency  int
	MaxAttempts  int
	PollInterval time.Duration
	LockDuration time.Duration
	SendTimeout  time.Duration
}

func NewWorker(store DeliveryStore, dispatcher Sender, logger *slog.Logger, metrics observability.Metrics, config WorkerConfig) *Worker {
	if metrics == nil {
		metrics = observability.NopMetrics{}
	}
	return &Worker{
		store: store, dispatcher: dispatcher, logger: logger,
		concurrency: config.Concurrency, maxAttempts: config.MaxAttempts,
		pollInterval: config.PollInterval, lockDuration: config.LockDuration, sendTimeout: config.SendTimeout, metrics: metrics,
	}
}

func (worker *Worker) Run(ctx context.Context) error {
	for ctx.Err() == nil {
		// Claim no more than can start immediately; every lease therefore exceeds the bounded send deadline.
		deliveries, err := worker.store.ClaimDeliveries(ctx, worker.concurrency, worker.lockDuration)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			worker.logger.Error("claim push deliveries", "error", err)
			if !wait(ctx, worker.pollInterval) {
				break
			}
			continue
		}
		if len(deliveries) == 0 {
			if !wait(ctx, worker.pollInterval) {
				break
			}
			continue
		}
		worker.metrics.ObserveWorkerBatch("push", len(deliveries))
		worker.deliverBatch(ctx, deliveries)
	}
	return ctx.Err()
}

func (worker *Worker) deliverBatch(ctx context.Context, deliveries []notification.Delivery) {
	var group sync.WaitGroup
	group.Add(len(deliveries))
	for _, delivery := range deliveries {
		go func() {
			defer group.Done()
			worker.deliver(ctx, delivery)
		}()
	}
	group.Wait()
}

func (worker *Worker) deliver(ctx context.Context, delivery notification.Delivery) {
	sendContext, cancel := context.WithTimeout(ctx, worker.sendTimeout)
	result, err := worker.dispatcher.Send(sendContext, delivery)
	cancel()
	if err == nil {
		if err := worker.store.CompleteDelivery(ctx, delivery.ID, delivery.LockToken, result.MessageID); err != nil {
			worker.logger.Error("complete push delivery", "delivery_id", delivery.ID, "error", err)
			worker.metrics.ObserveWorkerResult("push", "lease_error")
			return
		}
		if delivery.Kind == notification.DeliveryLiveActivityEnd {
			if err := worker.store.InvalidateEndpoint(ctx, delivery.EndpointID, "live activity ended"); err != nil {
				worker.logger.Error("retire Live Activity endpoint", "endpoint_id", delivery.EndpointID, "error", err)
			}
		}
		worker.metrics.ObserveWorkerResult("push", "sent")
		return
	}
	if ctx.Err() != nil {
		worker.metrics.ObserveWorkerResult("push", "cancelled")
		return
	}
	if result.Reason == "" {
		result.Reason = err.Error()
	}
	terminal := result.Invalid || !result.Retryable || delivery.Attempts >= worker.maxAttempts
	backoff := time.Duration(math.Pow(2, float64(min(delivery.Attempts, 8)))) * time.Second
	if retryErr := worker.store.RetryDelivery(ctx, delivery.ID, delivery.LockToken, time.Now().Add(backoff), result.Reason, terminal); retryErr != nil {
		worker.logger.Error("reschedule push delivery", "delivery_id", delivery.ID, "error", retryErr)
		worker.metrics.ObserveWorkerResult("push", "lease_error")
		return
	}
	if terminal {
		worker.metrics.ObserveWorkerResult("push", "failed")
	} else {
		worker.metrics.ObserveWorkerResult("push", "retry")
	}
	if result.Invalid {
		if invalidateErr := worker.store.InvalidateEndpoint(ctx, delivery.EndpointID, result.Reason); invalidateErr != nil {
			worker.logger.Error("invalidate push endpoint", "endpoint_id", delivery.EndpointID, "error", invalidateErr)
		}
	}
	worker.logger.Warn("push delivery failed", "delivery_id", delivery.ID, "transport", delivery.Transport, "attempt", delivery.Attempts, "terminal", terminal, "error", err)
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
