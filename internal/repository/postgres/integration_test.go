package postgres_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pabloadvisory/ssb-backend/internal/demo"
	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
	"github.com/pabloadvisory/ssb-backend/internal/domain/news"
	"github.com/pabloadvisory/ssb-backend/internal/eventing"
	"github.com/pabloadvisory/ssb-backend/internal/notification"
	postgresrepo "github.com/pabloadvisory/ssb-backend/internal/repository/postgres"
	"github.com/pabloadvisory/ssb-backend/internal/service"
	"github.com/pabloadvisory/ssb-backend/migrations"
)

func TestPostgresIntegration(t *testing.T) {
	pool := newTestDatabase(t)
	ctx := context.Background()

	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("reapply migrations: %v", err)
	}
	var migrationCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatal(err)
	}
	if migrationCount != 5 {
		t.Fatalf("expected 5 forward migrations, got %d", migrationCount)
	}
	assertColumnAbsent(t, pool, "matches", "provider")
	assertColumnAbsent(t, pool, "push_endpoints", "token")

	if err := demo.Seed(ctx, pool); err != nil {
		t.Fatalf("seed demo data: %v", err)
	}
	repository := postgresrepo.New(pool)

	t.Run("news publication visibility and idempotency", func(t *testing.T) {
		newsService := service.NewNews(repository)
		seeded, err := newsService.ListPublishedArticles(ctx, news.Filter{Limit: 20})
		if err != nil {
			t.Fatal(err)
		}
		if len(seeded) != 2 {
			t.Fatalf("expected two published demo articles, got %d", len(seeded))
		}
		if _, err := newsService.GetPublishedArticleBySlug(ctx, "transfer-window-notes"); !errors.Is(err, news.ErrNotFound) {
			t.Fatalf("draft article must not be public, got %v", err)
		}

		publishedAt := time.Now().UTC().Add(-time.Minute)
		command := news.UpsertArticle{
			Slug: "integration-news", Title: "Integration news", Summary: "Repository test article",
			BodyMarkdown: "A published integration article.", Category: news.CategoryAnnouncement,
			Status: news.StatusPublished, PublishedAt: &publishedAt,
		}
		created, err := newsService.UpsertArticle(ctx, "integration", "news-1", command)
		if err != nil {
			t.Fatal(err)
		}
		unchanged, err := newsService.UpsertArticle(ctx, "integration", "news-1", command)
		if err != nil {
			t.Fatal(err)
		}
		if created.ID != unchanged.ID || created.Version != 1 || unchanged.Version != 1 {
			t.Fatalf("unchanged editorial retries must preserve identity and version: created=%+v unchanged=%+v", created, unchanged)
		}
		command.Title = "Updated integration news"
		updated, err := newsService.UpsertArticle(ctx, "integration", "news-1", command)
		if err != nil {
			t.Fatal(err)
		}
		if updated.ID != created.ID || updated.Version != 2 {
			t.Fatalf("changed article should preserve ID and increment version: %+v", updated)
		}
		command.Status = news.StatusArchived
		archived, err := newsService.UpsertArticle(ctx, "integration", "news-1", command)
		if err != nil {
			t.Fatal(err)
		}
		if archived.Version != 3 {
			t.Fatalf("archived article should increment version: %+v", archived)
		}
		if _, err := newsService.GetPublishedArticleBySlug(ctx, command.Slug); !errors.Is(err, news.ErrNotFound) {
			t.Fatalf("archived article must no longer be public, got %v", err)
		}

		future := time.Now().UTC().Add(time.Hour)
		command.Slug = "scheduled-integration-news"
		command.Status = news.StatusPublished
		command.PublishedAt = &future
		if _, err := newsService.UpsertArticle(ctx, "integration", "news-2", command); err != nil {
			t.Fatal(err)
		}
		if _, err := newsService.GetPublishedArticleBySlug(ctx, command.Slug); !errors.Is(err, news.ErrNotFound) {
			t.Fatalf("scheduled article must stay hidden before publication, got %v", err)
		}
	})

	t.Run("hash gated version and outbox ordering", func(t *testing.T) {
		footballService := service.NewFootball(repository)
		homeScore, awayScore := int16(0), int16(0)
		snapshot := football.MatchSnapshot{
			LeagueID: "10000000-0000-0000-0000-000000000001", SeasonID: "10000000-0000-0000-0000-000000000002",
			Round: stringPointer("Integration round"), RoundSort: intPointer(99), Leg: 1, KickoffAt: time.Now().UTC().Add(time.Hour),
			Status: football.MatchScheduled, HomeTeamID: "20000000-0000-0000-0000-000000000001",
			AwayTeamID: "20000000-0000-0000-0000-000000000003", HomeScore: &homeScore, AwayScore: &awayScore,
		}
		created, err := footballService.UpsertMatchSnapshot(ctx, "integration", "ordered-match", snapshot)
		if err != nil {
			t.Fatal(err)
		}
		unchanged, err := footballService.UpsertMatchSnapshot(ctx, "integration", "ordered-match", snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if created.Version != 1 || unchanged.Version != 1 {
			t.Fatalf("unchanged source must preserve version 1: created=%d unchanged=%d", created.Version, unchanged.Version)
		}
		homeScore = 1
		updated, err := footballService.UpsertMatchSnapshot(ctx, "integration", "ordered-match", snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if updated.ID != created.ID || updated.Version != 2 {
			t.Fatalf("expected stable canonical ID and version 2, got id=%s version=%d", updated.ID, updated.Version)
		}
		var mappedID string
		if err := pool.QueryRow(ctx, `
			SELECT entity_id FROM external_ids
			WHERE provider='integration' AND entity_type='match' AND external_id='ordered-match'`).Scan(&mappedID); err != nil {
			t.Fatal(err)
		}
		if mappedID != created.ID {
			t.Fatalf("external identity maps to %s, expected %s", mappedID, created.ID)
		}
		var eventCount int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events WHERE aggregate_id=$1`, created.ID).Scan(&eventCount); err != nil {
			t.Fatal(err)
		}
		if eventCount != 2 {
			t.Fatalf("expected exactly two changed snapshots in outbox, got %d", eventCount)
		}

		first, err := repository.ClaimOutboxEvents(ctx, 10, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if len(first) != 1 {
			t.Fatalf("per-aggregate ordering should claim one event, got %d", len(first))
		}
		var changed eventing.MatchChanged
		if err := json.Unmarshal(first[0].Payload, &changed); err != nil {
			t.Fatal(err)
		}
		if err := repository.PublishMatchChanged(ctx, first[0], changed, notification.PlanMatchDeliveries(changed.Notification)); err != nil {
			t.Fatal(err)
		}
		second, err := repository.ClaimOutboxEvents(ctx, 10, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		if len(second) != 1 || second[0].ID == first[0].ID {
			t.Fatalf("expected the next aggregate event after terminal completion, got %+v", second)
		}
		if err := repository.RetryOutboxEvent(ctx, second[0].ID, second[0].LockToken, time.Now(), "terminal test", true); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delivery lease retry and isolated token", func(t *testing.T) {
		notifications := service.NewNotifications(repository)
		installation, err := notifications.CreateInstallation(ctx, notification.CreateInstallation{
			Platform: notification.PlatformAndroid, AppID: "com.pabloadvisory.ssb.integration",
		})
		if err != nil {
			t.Fatal(err)
		}
		endpoint, err := notifications.RegisterEndpoint(ctx, installation.ID, installation.Credential, notification.EndpointStandard, notification.RegisterEndpoint{
			Transport: notification.TransportFCM, Token: "integration-secret-device-token", Environment: "production",
		})
		if err != nil {
			t.Fatal(err)
		}
		var storedToken string
		if err := pool.QueryRow(ctx, `SELECT token FROM push_endpoint_tokens WHERE endpoint_id=$1`, endpoint.ID).Scan(&storedToken); err != nil {
			t.Fatal(err)
		}
		if storedToken != "integration-secret-device-token" {
			t.Fatal("push token was not stored in the isolated credential table")
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO notification_deliveries (
				endpoint_id, match_id, kind, payload, priority, idempotency_key
			) VALUES ($1, '40000000-0000-0000-0000-000000000001', 'match_update',
				'{"match_id":"40000000-0000-0000-0000-000000000001","version":7,"status":"live"}',
				'high', 'integration-lease')`, endpoint.ID); err != nil {
			t.Fatal(err)
		}
		claimed, err := repository.ClaimDeliveries(ctx, 1, time.Minute)
		if err != nil || len(claimed) != 1 {
			t.Fatalf("claim first lease: deliveries=%d error=%v", len(claimed), err)
		}
		firstLease := claimed[0]
		if _, err := pool.Exec(ctx, `UPDATE notification_deliveries SET locked_until=now()-interval '1 second' WHERE id=$1`, firstLease.ID); err != nil {
			t.Fatal(err)
		}
		reclaimed, err := repository.ClaimDeliveries(ctx, 1, time.Minute)
		if err != nil || len(reclaimed) != 1 {
			t.Fatalf("reclaim expired lease: deliveries=%d error=%v", len(reclaimed), err)
		}
		if reclaimed[0].LockToken == firstLease.LockToken {
			t.Fatal("reclaimed delivery must receive a new lease token")
		}
		if err := repository.CompleteDelivery(ctx, firstLease.ID, firstLease.LockToken, "stale"); !errors.Is(err, notification.ErrLeaseLost) {
			t.Fatalf("stale worker should lose lease, got %v", err)
		}
		if err := repository.RetryDelivery(ctx, reclaimed[0].ID, reclaimed[0].LockToken, time.Now().Add(-time.Second), "retry", false); err != nil {
			t.Fatal(err)
		}
		retried, err := repository.ClaimDeliveries(ctx, 1, time.Minute)
		if err != nil || len(retried) != 1 || retried[0].Attempts != 3 {
			t.Fatalf("claim retried delivery: %+v error=%v", retried, err)
		}
		if err := repository.CompleteDelivery(ctx, retried[0].ID, retried[0].LockToken, "provider-message"); err != nil {
			t.Fatal(err)
		}
	})

	queues, err := repository.QueueHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(queues) != 2 {
		t.Fatalf("expected both queue metrics, got %+v", queues)
	}
}

func newTestDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("SSB_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("SSB_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public`); err != nil {
		admin.Close()
		t.Fatalf("prepare pg_trgm extension: %v", err)
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	schema := "ssb_test_" + hex.EncodeToString(random)
	identifier := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		admin.Close()
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+identifier+" CASCADE")
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if !strings.HasPrefix(schema, "ssb_test_") || len(schema) != len("ssb_test_")+16 {
			t.Errorf("refusing to clean unexpected test schema %q", schema)
			admin.Close()
			return
		}
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Errorf("drop test schema %s: %v", schema, err)
		}
		admin.Close()
	})
	return pool
}

func assertColumnAbsent(t *testing.T, pool *pgxpool.Pool, table, column string) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema=current_schema() AND table_name=$1 AND column_name=$2
		)`, table, column).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("%s.%s should not exist", table, column)
	}
}

func stringPointer(value string) *string { return &value }
func intPointer(value int) *int          { return &value }
