package push

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/observability"
)

type workerStore struct {
	mu          sync.Mutex
	cancel      context.CancelFunc
	claimCalls  int
	claimLimit  int
	completions int
}

func (store *workerStore) ClaimDeliveries(_ context.Context, limit int, _ time.Duration) ([]notification.Delivery, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.claimCalls++
	if store.claimCalls > 1 {
		store.cancel()
		return nil, context.Canceled
	}
	store.claimLimit = limit
	deliveries := make([]notification.Delivery, limit)
	for index := range deliveries {
		deliveries[index] = notification.Delivery{ID: string(rune('a' + index)), LockToken: "lease"}
	}
	return deliveries, nil
}

func (store *workerStore) CompleteDelivery(context.Context, string, string, string) error {
	store.mu.Lock()
	store.completions++
	store.mu.Unlock()
	return nil
}

func (*workerStore) RetryDelivery(context.Context, string, string, time.Time, string, bool) error {
	return nil
}

func (*workerStore) InvalidateEndpoint(context.Context, string, string) error { return nil }

type successfulSender struct{}

func (successfulSender) Send(context.Context, notification.Delivery) (Result, error) {
	time.Sleep(5 * time.Millisecond)
	return Result{MessageID: "sent"}, nil
}

func TestWorkerClaimsOnlyImmediatelyRunnableDeliveries(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	store := &workerStore{cancel: cancel}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	worker := NewWorker(store, successfulSender{}, logger, observability.NopMetrics{}, WorkerConfig{
		Concurrency: 3, MaxAttempts: 3,
		PollInterval: time.Millisecond, LockDuration: 30 * time.Second, SendTimeout: time.Second,
	})
	_ = worker.Run(ctx)

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.claimLimit != 3 {
		t.Fatalf("expected claim limit 3, got %d", store.claimLimit)
	}
	if store.completions != 3 {
		t.Fatalf("expected 3 completions, got %d", store.completions)
	}
}
