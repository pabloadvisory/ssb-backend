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
	if migrationCount != 7 {
		t.Fatalf("expected 7 forward migrations, got %d", migrationCount)
	}
	assertColumnAbsent(t, pool, "matches", "provider")
	assertColumnAbsent(t, pool, "push_endpoints", "token")

	if err := demo.Seed(ctx, pool); err != nil {
		t.Fatalf("seed demo data: %v", err)
	}
	repository := postgresrepo.New(pool)

	t.Run("public football coverage and discovery reads", func(t *testing.T) {
		footballService := service.NewFootball(repository)
		const liveMatchID = "40000000-0000-0000-0000-000000000001"
		const leagueID = "10000000-0000-0000-0000-000000000001"
		const seasonID = "10000000-0000-0000-0000-000000000002"
		const venueID = "10000000-0000-0000-0000-000000000003"
		const teamAID = "20000000-0000-0000-0000-000000000002"
		const teamBID = "20000000-0000-0000-0000-000000000003"

		league, err := footballService.GetLeague(ctx, leagueID)
		if err != nil {
			t.Fatal(err)
		}
		if league.CurrentSeasonID == nil || *league.CurrentSeasonID != seasonID {
			t.Fatalf("league did not expose its current season: %+v", league)
		}
		seasons, err := footballService.ListLeagueSeasons(ctx, leagueID)
		if err != nil {
			t.Fatal(err)
		}
		if len(seasons.Data) != 1 || seasons.Data[0].ID != seasonID || !seasons.Data[0].IsCurrent {
			t.Fatalf("unexpected league seasons: %+v", seasons)
		}

		lineups, err := footballService.GetMatchLineups(ctx, liveMatchID)
		if err != nil {
			t.Fatal(err)
		}
		if lineups.Home.Team.ID == "" || lineups.Away.Team.ID == "" ||
			len(lineups.Home.Starters) != 11 || len(lineups.Home.Substitutes) != 7 ||
			len(lineups.Away.Starters) != 11 || len(lineups.Away.Substitutes) != 7 {
			t.Fatalf("unexpected seeded lineups: %+v", lineups)
		}
		if lineups.Home.Coach == nil || lineups.Home.Coach.DisplayName == "" ||
			lineups.Away.Coach == nil || lineups.Away.Coach.DisplayName == "" {
			t.Fatalf("both seeded lineups must include managers: %+v", lineups)
		}
		positions := make(map[string]bool)
		for _, player := range lineups.Home.Starters {
			if player.Player.Position != nil {
				positions[*player.Player.Position] = true
			}
		}
		for _, position := range []string{"goalkeeper", "defender", "midfielder", "forward"} {
			if !positions[position] {
				t.Fatalf("home starting lineup is missing %s coverage: %+v", position, lineups.Home.Starters)
			}
		}
		var alexStatus, liamStatus string
		for _, player := range lineups.Home.Starters {
			if player.Player.ID == "30000000-0000-0000-0000-000000000001" {
				alexStatus = player.SubstitutionStatus
			}
		}
		for _, player := range lineups.Home.Substitutes {
			if player.Player.ID == "30000000-0000-0000-0000-000000000009" {
				liamStatus = player.SubstitutionStatus
			}
		}
		if alexStatus != "substituted_out" || liamStatus != "substituted_in" {
			t.Fatalf("substitution direction was not derived correctly: home=%+v", lineups.Home)
		}

		statistics, err := footballService.GetMatchStatistics(ctx, liveMatchID)
		if err != nil {
			t.Fatal(err)
		}
		if statistics.Home.Totals == nil || statistics.Away.Totals == nil || statistics.Home.Totals.Possession == nil || len(statistics.Home.Players) < 2 {
			t.Fatalf("unexpected seeded match statistics: %+v", statistics)
		}

		standings, err := footballService.ListSeasonStandings(ctx, seasonID)
		if err != nil {
			t.Fatal(err)
		}
		if len(standings.Data) != 4 || standings.Data[0].GoalDifference != 15 || standings.Data[0].HomeRecord == nil {
			t.Fatalf("unexpected seeded standings: %+v", standings)
		}

		venue, err := footballService.GetVenue(ctx, venueID)
		if err != nil {
			t.Fatal(err)
		}
		if venue.CountryName == nil || venue.Latitude == nil || venue.Longitude == nil || venue.Surface == nil {
			t.Fatalf("venue detail is incomplete: %+v", venue)
		}

		officials, err := footballService.ListMatchOfficials(ctx, liveMatchID)
		if err != nil {
			t.Fatal(err)
		}
		if len(officials.Data) != 1 || officials.Data[0].Role != football.OfficialReferee || officials.Data[0].Person.DisplayName == "" {
			t.Fatalf("unexpected officials: %+v", officials)
		}

		results, err := footballService.Search(ctx, football.SearchFilter{Query: "victoria", Limit: 20})
		if err != nil {
			t.Fatal(err)
		}
		if len(results) == 0 || results[0].EntityType != football.SearchTeam {
			t.Fatalf("unexpected search results: %+v", results)
		}
		discoveryResults, err := footballService.Search(ctx, football.SearchFilter{
			Query: "premier", Types: []football.SearchEntityType{football.SearchLeague, football.SearchFixture}, Limit: 20,
		})
		if err != nil {
			t.Fatal(err)
		}
		var foundLeague, foundFixture bool
		for _, result := range discoveryResults {
			foundLeague = foundLeague || result.EntityType == football.SearchLeague
			foundFixture = foundFixture || result.EntityType == football.SearchFixture
		}
		if !foundLeague || !foundFixture {
			t.Fatalf("search did not include league and fixture results: %+v", discoveryResults)
		}

		headToHead, err := footballService.HeadToHead(ctx, football.HeadToHeadFilter{TeamAID: teamAID, TeamBID: teamBID, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if headToHead.Summary.TeamAWins != 1 || headToHead.Summary.Draws != 1 || headToHead.Summary.TeamBWins != 1 || len(headToHead.Meetings) != 3 {
			t.Fatalf("unexpected head-to-head summary: %+v", headToHead)
		}
	})

	t.Run("installation prediction is unique and mutable before kickoff", func(t *testing.T) {
		notifications := service.NewNotifications(repository)
		installation, err := notifications.CreateInstallation(ctx, notification.CreateInstallation{
			Platform: notification.PlatformIOS, AppID: "com.pabloadvisory.ssb.prediction",
		})
		if err != nil {
			t.Fatal(err)
		}
		footballService := service.NewFootball(repository)
		const matchID = "40000000-0000-0000-0000-000000000002"
		if _, err := pool.Exec(ctx, `UPDATE matches SET kickoff_at=$2 WHERE id=$1`, matchID, time.Now().UTC().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		first, err := footballService.SetInstallationPrediction(ctx, installation.ID, installation.Credential, matchID, football.PredictionHome)
		if err != nil {
			t.Fatal(err)
		}
		updated, err := footballService.SetInstallationPrediction(ctx, installation.ID, installation.Credential, matchID, football.PredictionAway)
		if err != nil {
			t.Fatal(err)
		}
		if first.TotalVotes != 1 || updated.TotalVotes != 1 || updated.MySelection == nil || *updated.MySelection != football.PredictionAway {
			t.Fatalf("vote update must retain one voter and change selection: first=%+v updated=%+v", first, updated)
		}
	})

	t.Run("coverage replacement distinguishes supplied empty datasets", func(t *testing.T) {
		footballService := service.NewFootball(repository)
		const matchID = "40000000-0000-0000-0000-000000000002"
		lineups := []football.LineupInput{
			{
				TeamID: "20000000-0000-0000-0000-000000000003", PersonID: "30000000-0000-0000-0000-000000000003",
				IsStarter: true,
			},
			{
				TeamID: "20000000-0000-0000-0000-000000000004", PersonID: "30000000-0000-0000-0000-000000000004",
				IsStarter: true,
			},
		}
		if err := footballService.ReplaceMatchCoverage(ctx, matchID, "integration", football.MatchCoverageUpdate{Lineups: &lineups}); err != nil {
			t.Fatal(err)
		}
		stored, err := footballService.GetMatchLineups(ctx, matchID)
		if err != nil {
			t.Fatal(err)
		}
		if len(stored.Home.Starters) != 1 || len(stored.Away.Starters) != 1 {
			t.Fatalf("coverage replacement did not store both sides: %+v", stored)
		}
		empty := []football.LineupInput{}
		if err := footballService.ReplaceMatchCoverage(ctx, matchID, "integration", football.MatchCoverageUpdate{Lineups: &empty}); err != nil {
			t.Fatal(err)
		}
		cleared, err := footballService.GetMatchLineups(ctx, matchID)
		if err != nil {
			t.Fatal(err)
		}
		if len(cleared.Home.Starters) != 0 || len(cleared.Away.Starters) != 0 {
			t.Fatalf("explicit empty lineup dataset did not clear rows: %+v", cleared)
		}
	})

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
