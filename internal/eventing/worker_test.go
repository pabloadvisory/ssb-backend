package eventing

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/observability"
	"github.com/pabloadvisory/ssb-backend/internal/realtime"
)

type eventStore struct {
	mu         sync.Mutex
	cancel     context.CancelFunc
	claimCalls int
	claimLimit int
	published  int
	payload    json.RawMessage
}

func (store *eventStore) ClaimOutboxEvents(_ context.Context, limit int, _ time.Duration) ([]Event, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.claimCalls++
	if store.claimCalls > 1 {
		store.cancel()
		return nil, context.Canceled
	}
	store.claimLimit = limit
	return []Event{
		{ID: "one", Type: MatchChangedV1, Payload: store.payload, LockToken: "lease-one", Attempts: 1},
		{ID: "two", Type: MatchChangedV1, Payload: store.payload, LockToken: "lease-two", Attempts: 1},
	}, nil
}

func (store *eventStore) PublishMatchChanged(context.Context, Event, MatchChanged, []notification.DeliveryPlan) error {
	store.mu.Lock()
	store.published++
	store.mu.Unlock()
	return nil
}

func (*eventStore) RetryOutboxEvent(context.Context, string, string, time.Time, string, bool) error {
	return nil
}

func TestWorkerProcessesOnlyImmediatelyRunnableBatch(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	payload, err := json.Marshal(MatchChanged{
		Realtime: realtime.Update{MatchID: "match-1", Version: 1},
		Notification: notification.MatchChange{
			Current: notification.MatchUpdate{MatchID: "match-1", Version: 1, Status: "scheduled"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	store := &eventStore{cancel: cancel, payload: payload}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	worker := NewWorker(store, logger, observability.NopMetrics{}, WorkerConfig{
		Concurrency: 2, MaxAttempts: 3,
		PollInterval: time.Millisecond, LockDuration: 30 * time.Second, ProcessTimeout: time.Second,
	})
	worker.Run(ctx)

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.published != 2 {
		t.Fatalf("expected 2 published events, got %d", store.published)
	}
	if store.claimLimit != 2 {
		t.Fatalf("expected claim limit 2, got %d", store.claimLimit)
	}
}
