package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/pabloadvisory/ssb-backend/internal/demo"
	"github.com/pabloadvisory/ssb-backend/internal/notification"
	postgresrepo "github.com/pabloadvisory/ssb-backend/internal/repository/postgres"
	"github.com/pabloadvisory/ssb-backend/internal/service"
	"github.com/pabloadvisory/ssb-backend/migrations"
)

func TestPlayerFollowsAndNotificationPreferencesIntegration(t *testing.T) {
	pool := newTestDatabase(t)
	ctx := context.Background()
	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := demo.Seed(ctx, pool); err != nil {
		t.Fatalf("seed demo data: %v", err)
	}

	repository := postgresrepo.New(pool)
	notifications := service.NewNotifications(repository)
	installation, err := notifications.CreateInstallation(ctx, notification.CreateInstallation{
		Platform: notification.PlatformIOS,
		AppID:    "com.pabloadvisory.ssb.player-follows-test",
	})
	if err != nil {
		t.Fatal(err)
	}

	preferences, err := notifications.GetNotificationPreferences(ctx, installation.ID, installation.Credential)
	if err != nil {
		t.Fatal(err)
	}
	if !preferences.MatchUpdatesEnabled || !preferences.MatchFinishedEnabled || !preferences.FollowedPlayerEventsEnabled {
		t.Fatalf("expected default preferences to be enabled, got %+v", preferences)
	}

	const playerID = "30000000-0000-0000-0000-000000000001"
	follow, err := notifications.SetPlayerFollow(ctx, installation.ID, installation.Credential, playerID, notification.SetPlayerFollow{
		NotificationsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if follow.Player.ID != playerID || !follow.NotificationsEnabled {
		t.Fatalf("unexpected follow: %+v", follow)
	}
	repeated, err := notifications.SetPlayerFollow(ctx, installation.ID, installation.Credential, playerID, notification.SetPlayerFollow{
		NotificationsEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repeated.FollowedAt.Equal(follow.FollowedAt) || !repeated.UpdatedAt.Equal(follow.UpdatedAt) {
		t.Fatalf("idempotent follow changed timestamps: first=%+v repeated=%+v", follow, repeated)
	}

	follows, err := notifications.ListPlayerFollows(ctx, installation.ID, installation.Credential, notification.PlayerFollowFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(follows) != 1 || follows[0].Player.ID != playerID {
		t.Fatalf("unexpected follows: %+v", follows)
	}

	updatedPreferences, err := notifications.SetNotificationPreferences(ctx, installation.ID, installation.Credential, notification.SetNotificationPreferences{
		MatchUpdatesEnabled:         false,
		MatchFinishedEnabled:        true,
		FollowedPlayerEventsEnabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	repeatedPreferences, err := notifications.SetNotificationPreferences(ctx, installation.ID, installation.Credential, notification.SetNotificationPreferences{
		MatchUpdatesEnabled:         false,
		MatchFinishedEnabled:        true,
		FollowedPlayerEventsEnabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !repeatedPreferences.UpdatedAt.Equal(updatedPreferences.UpdatedAt) {
		t.Fatalf("idempotent preferences update changed timestamp: first=%+v repeated=%+v", updatedPreferences, repeatedPreferences)
	}

	if err := notifications.DeletePlayerFollow(ctx, installation.ID, installation.Credential, playerID); err != nil {
		t.Fatal(err)
	}
	if err := notifications.DeletePlayerFollow(ctx, installation.ID, installation.Credential, playerID); err != nil {
		t.Fatalf("repeated unfollow must be idempotent: %v", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE app_installations SET disabled_at = now() WHERE id = $1`, installation.ID); err != nil {
		t.Fatal(err)
	}
	_, err = notifications.SetPlayerFollow(
		ctx,
		installation.ID,
		installation.Credential,
		"30000000-0000-0000-0000-000000000099",
		notification.SetPlayerFollow{NotificationsEnabled: true},
	)
	if !errors.Is(err, notification.ErrUnauthorized) {
		t.Fatalf("disabled installation must be rejected before player lookup, got %v", err)
	}
}
