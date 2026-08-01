package postgres

import (
	"context"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/observability"
)

func (store *Store) QueueHealth(ctx context.Context) ([]observability.QueueHealth, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT 'outbox', count(*), COALESCE(EXTRACT(epoch FROM now() - min(occurred_at)), 0)
		FROM outbox_events WHERE published_at IS NULL AND failed_at IS NULL
		UNION ALL
		SELECT 'notification_deliveries', count(*), COALESCE(EXTRACT(epoch FROM now() - min(created_at)), 0)
		FROM notification_deliveries WHERE state IN ('pending', 'sending')`)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	queues := make([]observability.QueueHealth, 0, 2)
	for rows.Next() {
		var queue observability.QueueHealth
		var seconds float64
		if err := rows.Scan(&queue.Queue, &queue.Depth, &seconds); err != nil {
			return nil, mapError(err)
		}
		queue.OldestAge = time.Duration(seconds * float64(time.Second))
		queues = append(queues, queue)
	}
	return queues, mapError(rows.Err())
}

var _ observability.QueueStore = (*Store)(nil)
