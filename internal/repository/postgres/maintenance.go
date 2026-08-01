package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pabloadvisory/ssb-backend/internal/maintenance"
)

func (store *Store) Prune(
	ctx context.Context,
	deliveryBefore, outboxBefore, auditBefore time.Time,
	limit int,
) (maintenance.Result, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return maintenance.Result{}, fmt.Errorf("begin retention cleanup: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var result maintenance.Result
	deliveryResult, err := tx.Exec(ctx, `
		WITH targets AS (
			SELECT id FROM notification_deliveries
			WHERE state IN ('sent', 'failed') AND completed_at < $1
			ORDER BY completed_at, id FOR UPDATE SKIP LOCKED LIMIT $2
		)
		DELETE FROM notification_deliveries delivery
		USING targets WHERE delivery.id = targets.id`, deliveryBefore, limit)
	if err != nil {
		return maintenance.Result{}, mapError(err)
	}
	result.Deliveries = deliveryResult.RowsAffected()

	outboxResult, err := tx.Exec(ctx, `
		WITH targets AS (
			SELECT id FROM outbox_events
			WHERE (published_at IS NOT NULL AND published_at < $1)
			   OR (failed_at IS NOT NULL AND failed_at < $1)
			ORDER BY COALESCE(published_at, failed_at), id FOR UPDATE SKIP LOCKED LIMIT $2
		)
		DELETE FROM outbox_events event
		USING targets WHERE event.id = targets.id`, outboxBefore, limit)
	if err != nil {
		return maintenance.Result{}, mapError(err)
	}
	result.Outbox = outboxResult.RowsAffected()

	auditResult, err := tx.Exec(ctx, `
		WITH targets AS (
			SELECT id FROM push_endpoint_registration_audit
			WHERE occurred_at < $1
			ORDER BY occurred_at, id FOR UPDATE SKIP LOCKED LIMIT $2
		)
		DELETE FROM push_endpoint_registration_audit audit
		USING targets WHERE audit.id = targets.id`, auditBefore, limit)
	if err != nil {
		return maintenance.Result{}, mapError(err)
	}
	result.Audit = auditResult.RowsAffected()

	if err := tx.Commit(ctx); err != nil {
		return maintenance.Result{}, mapError(err)
	}
	return result, nil
}

var _ maintenance.Store = (*Store)(nil)
