package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pabloadvisory/ssb-backend/internal/observability"
)

const Channel = "ssb_match_updates"

type Update struct {
	MatchID string `json:"match_id"`
	Type    string `json:"type"`
	Version int64  `json:"version"`
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[uint64]chan Update
	latest      map[string]int64
	nextID      uint64
	metrics     observability.Metrics
}

func NewHub(metricAdapters ...observability.Metrics) *Hub {
	var metrics observability.Metrics = observability.NopMetrics{}
	if len(metricAdapters) > 0 && metricAdapters[0] != nil {
		metrics = metricAdapters[0]
	}
	return &Hub{subscribers: make(map[string]map[uint64]chan Update), latest: make(map[string]int64), metrics: metrics}
}

func (h *Hub) Subscribe(matchID string) (<-chan Update, func()) {
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	updates := make(chan Update, 1)
	if h.subscribers[matchID] == nil {
		h.subscribers[matchID] = make(map[uint64]chan Update)
	}
	h.subscribers[matchID][id] = updates
	h.mu.Unlock()
	h.metrics.AddRealtimeSubscribers(1)

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subscribers[matchID], id)
			if len(h.subscribers[matchID]) == 0 {
				delete(h.subscribers, matchID)
				delete(h.latest, matchID)
			}
			close(updates)
			h.mu.Unlock()
			h.metrics.AddRealtimeSubscribers(-1)
		})
	}
	return updates, cancel
}

func (h *Hub) Publish(update Update) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.subscribers[update.MatchID]) == 0 {
		return
	}
	if update.Version <= h.latest[update.MatchID] {
		return
	}
	h.latest[update.MatchID] = update.Version
	for _, subscriber := range h.subscribers[update.MatchID] {
		select {
		case subscriber <- update:
		default:
			// Keep memory bounded while replacing stale state with the newest match version.
			select {
			case <-subscriber:
			default:
			}
			select {
			case subscriber <- update:
			default:
			}
		}
	}
}

func (h *Hub) SubscribedMatchIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.subscribers))
	for matchID := range h.subscribers {
		ids = append(ids, matchID)
	}
	sort.Strings(ids)
	return ids
}

type Listener struct {
	pool   *pgxpool.Pool
	hub    *Hub
	logger *slog.Logger
}

func NewListener(pool *pgxpool.Pool, hub *Hub, logger *slog.Logger) *Listener {
	return &Listener{pool: pool, hub: hub, logger: logger}
}

func (listener *Listener) Run(ctx context.Context) {
	delay := 100 * time.Millisecond
	for ctx.Err() == nil {
		if err := listener.listen(ctx); err != nil && ctx.Err() == nil {
			listener.logger.Error("realtime listener disconnected", "error", err, "retry_in", delay)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			if delay < 5*time.Second {
				delay *= 2
			}
			continue
		}
		delay = 100 * time.Millisecond
	}
}

func (listener *Listener) listen(ctx context.Context) error {
	connection, err := listener.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer connection.Release()

	if _, err := connection.Exec(ctx, "LISTEN "+Channel); err != nil {
		return err
	}
	if err := listener.reconcile(ctx); err != nil {
		return err
	}
	listener.logger.Info("realtime listener ready", "channel", Channel)

	for {
		notification, err := connection.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		var update Update
		if err := json.Unmarshal([]byte(notification.Payload), &update); err != nil {
			listener.logger.Warn("ignored malformed realtime notification", "error", err)
			continue
		}
		if update.MatchID == "" {
			continue
		}
		listener.hub.Publish(update)
	}
}

func (listener *Listener) reconcile(ctx context.Context) error {
	matchIDs := listener.hub.SubscribedMatchIDs()
	if len(matchIDs) == 0 {
		return nil
	}
	rows, err := listener.pool.Query(ctx, `
		SELECT id::text, version FROM matches WHERE id::text = ANY($1::text[])
		ORDER BY id`, matchIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var update Update
		if err := rows.Scan(&update.MatchID, &update.Version); err != nil {
			return err
		}
		update.Type = "match.updated"
		listener.hub.Publish(update)
	}
	return rows.Err()
}
