package observability

import (
	"context"
	"log/slog"
	"time"
)

// Metrics is intentionally vendor-neutral. A Prometheus, OpenTelemetry, or hosted
// metrics adapter can be added without coupling the HTTP, worker, or realtime layers.
type Metrics interface {
	ObserveHTTPRequest(method, route string, status int, duration time.Duration)
	ObserveWorkerBatch(worker string, claimed int)
	ObserveWorkerResult(worker, outcome string)
	SetQueueHealth(queue string, depth int64, oldestAge time.Duration)
	AddRealtimeSubscribers(delta int)
}

type NopMetrics struct{}

func (NopMetrics) ObserveHTTPRequest(string, string, int, time.Duration) {}
func (NopMetrics) ObserveWorkerBatch(string, int)                        {}
func (NopMetrics) ObserveWorkerResult(string, string)                    {}
func (NopMetrics) SetQueueHealth(string, int64, time.Duration)           {}
func (NopMetrics) AddRealtimeSubscribers(int)                            {}

type SlogMetrics struct {
	logger *slog.Logger
}

func NewSlog(logger *slog.Logger) *SlogMetrics {
	return &SlogMetrics{logger: logger}
}

func (metrics *SlogMetrics) ObserveHTTPRequest(method, route string, status int, duration time.Duration) {
	metrics.logger.Debug("http metric", "method", method, "route", route, "status", status, "duration", duration)
}

func (metrics *SlogMetrics) ObserveWorkerBatch(worker string, claimed int) {
	metrics.logger.Debug("worker batch metric", "worker", worker, "claimed", claimed)
}

func (metrics *SlogMetrics) ObserveWorkerResult(worker, outcome string) {
	metrics.logger.Debug("worker result metric", "worker", worker, "outcome", outcome)
}

func (metrics *SlogMetrics) SetQueueHealth(queue string, depth int64, oldestAge time.Duration) {
	metrics.logger.Info("queue health metric", "queue", queue, "depth", depth, "oldest_pending_age", oldestAge)
}

func (metrics *SlogMetrics) AddRealtimeSubscribers(delta int) {
	metrics.logger.Debug("realtime subscriber metric", "delta", delta)
}

type QueueHealth struct {
	Queue     string
	Depth     int64
	OldestAge time.Duration
}

type QueueStore interface {
	QueueHealth(context.Context) ([]QueueHealth, error)
}

type QueueMonitor struct {
	store    QueueStore
	metrics  Metrics
	logger   *slog.Logger
	interval time.Duration
}

func NewQueueMonitor(store QueueStore, metrics Metrics, logger *slog.Logger, interval time.Duration) *QueueMonitor {
	return &QueueMonitor{store: store, metrics: metrics, logger: logger, interval: interval}
}

func (monitor *QueueMonitor) Run(ctx context.Context) {
	monitor.observe(ctx)
	ticker := time.NewTicker(monitor.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			monitor.observe(ctx)
		}
	}
}

func (monitor *QueueMonitor) observe(ctx context.Context) {
	queues, err := monitor.store.QueueHealth(ctx)
	if err != nil {
		if ctx.Err() == nil {
			monitor.logger.Error("measure queue health", "error", err)
		}
		return
	}
	for _, queue := range queues {
		monitor.metrics.SetQueueHealth(queue.Queue, queue.Depth, queue.OldestAge)
	}
}
