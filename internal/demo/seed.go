package demo

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed seed.sql
var seedSQL string

func Seed(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin demo seed: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(731944842610377)`); err != nil {
		return fmt.Errorf("lock demo seed: %w", err)
	}
	if _, err := tx.Exec(ctx, seedSQL); err != nil {
		return fmt.Errorf("apply demo seed: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit demo seed: %w", err)
	}
	return nil
}
