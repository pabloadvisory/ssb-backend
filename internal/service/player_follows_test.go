package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/notification"
)

type playerFollowTestStore struct {
	notification.Store
	installationID string
	playerID       string
	secretHash     []byte
	filter         notification.PlayerFollowFilter
	command        notification.SetPlayerFollow
	preferences    notification.SetNotificationPreferences
}

func (store *playerFollowTestStore) SetPlayerFollow(
	_ context.Context,
	installationID string,
	secretHash []byte,
	playerID string,
	command notification.SetPlayerFollow,
) (notification.PlayerFollow, error) {
	store.installationID = installationID
	store.playerID = playerID
	store.secretHash = append([]byte(nil), secretHash...)
	store.command = command
	return notification.PlayerFollow{InstallationID: installationID, NotificationsEnabled: command.NotificationsEnabled}, nil
}

func (store *playerFollowTestStore) DeletePlayerFollow(_ context.Context, installationID string, secretHash []byte, playerID string) error {
	store.installationID = installationID
	store.playerID = playerID
	store.secretHash = append([]byte(nil), secretHash...)
	return nil
}

func (store *playerFollowTestStore) ListPlayerFollows(
	_ context.Context,
	installationID string,
	secretHash []byte,
	filter notification.PlayerFollowFilter,
) ([]notification.PlayerFollow, error) {
	store.installationID = installationID
	store.secretHash = append([]byte(nil), secretHash...)
	store.filter = filter
	return []notification.PlayerFollow{}, nil
}

func (store *playerFollowTestStore) GetNotificationPreferences(
	_ context.Context,
	installationID string,
	secretHash []byte,
) (notification.NotificationPreferences, error) {
	store.installationID = installationID
	store.secretHash = append([]byte(nil), secretHash...)
	return notification.NotificationPreferences{InstallationID: installationID}, nil
}

func (store *playerFollowTestStore) SetNotificationPreferences(
	_ context.Context,
	installationID string,
	secretHash []byte,
	command notification.SetNotificationPreferences,
) (notification.NotificationPreferences, error) {
	store.installationID = installationID
	store.secretHash = append([]byte(nil), secretHash...)
	store.preferences = command
	return notification.NotificationPreferences{InstallationID: installationID}, nil
}

func TestPlayerFollowMethodsHashInstallationCredential(t *testing.T) {
	t.Parallel()

	store := &playerFollowTestStore{}
	service := NewNotifications(store)
	command := notification.SetPlayerFollow{NotificationsEnabled: true}
	result, err := service.SetPlayerFollow(context.Background(), " installation-1 ", "credential", " player-1 ", command)
	if err != nil {
		t.Fatal(err)
	}
	expectedHash := sha256.Sum256([]byte("credential"))
	if store.installationID != "installation-1" || store.playerID != "player-1" || !bytes.Equal(store.secretHash, expectedHash[:]) {
		t.Fatalf("unexpected store arguments: installation=%q player=%q hash=%x", store.installationID, store.playerID, store.secretHash)
	}
	if !result.NotificationsEnabled || store.command != command {
		t.Fatalf("unexpected follow result or command: result=%+v command=%+v", result, store.command)
	}

	if err := service.DeletePlayerFollow(context.Background(), "installation-1", "credential", "player-1"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(store.secretHash, expectedHash[:]) {
		t.Fatalf("delete used an unexpected credential hash: %x", store.secretHash)
	}
}

func TestListPlayerFollowsDefaultsLimitAndValidatesCursor(t *testing.T) {
	t.Parallel()

	store := &playerFollowTestStore{}
	service := NewNotifications(store)
	if _, err := service.ListPlayerFollows(context.Background(), "installation-1", "credential", notification.PlayerFollowFilter{}); err != nil {
		t.Fatal(err)
	}
	if store.filter.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", store.filter.Limit)
	}

	before := time.Now()
	_, err := service.ListPlayerFollows(context.Background(), "installation-1", "credential", notification.PlayerFollowFilter{
		BeforeFollowedAt: &before,
	})
	if !errors.Is(err, notification.ErrInvalid) {
		t.Fatalf("expected invalid partial cursor, got %v", err)
	}
	_, err = service.ListPlayerFollows(context.Background(), "installation-1", "credential", notification.PlayerFollowFilter{Limit: 101})
	if !errors.Is(err, notification.ErrInvalid) {
		t.Fatalf("expected invalid limit, got %v", err)
	}
}

func TestNotificationPreferenceMethodsHashInstallationCredential(t *testing.T) {
	t.Parallel()

	store := &playerFollowTestStore{}
	service := NewNotifications(store)
	command := notification.SetNotificationPreferences{
		MatchUpdatesEnabled:         true,
		MatchFinishedEnabled:        false,
		FollowedPlayerEventsEnabled: true,
	}
	if _, err := service.SetNotificationPreferences(context.Background(), "installation-1", "credential", command); err != nil {
		t.Fatal(err)
	}
	expectedHash := sha256.Sum256([]byte("credential"))
	if store.preferences != command || !bytes.Equal(store.secretHash, expectedHash[:]) {
		t.Fatalf("unexpected preference store arguments: command=%+v hash=%x", store.preferences, store.secretHash)
	}
	if _, err := service.GetNotificationPreferences(context.Background(), "installation-1", "credential"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(store.secretHash, expectedHash[:]) {
		t.Fatalf("get used an unexpected credential hash: %x", store.secretHash)
	}
}
