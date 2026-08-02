package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/pabloadvisory/ssb-backend/internal/notification"
)

func (service *Notifications) SetPlayerFollow(
	ctx context.Context,
	installationID, credential, playerID string,
	command notification.SetPlayerFollow,
) (notification.PlayerFollow, error) {
	installationID = strings.TrimSpace(installationID)
	playerID = strings.TrimSpace(playerID)
	if installationID == "" || playerID == "" {
		return notification.PlayerFollow{}, fmt.Errorf("%w: installation_id and player_id are required", notification.ErrInvalid)
	}
	credentialHash := sha256.Sum256([]byte(credential))
	return service.store.SetPlayerFollow(ctx, installationID, credentialHash[:], playerID, command)
}

func (service *Notifications) DeletePlayerFollow(ctx context.Context, installationID, credential, playerID string) error {
	installationID = strings.TrimSpace(installationID)
	playerID = strings.TrimSpace(playerID)
	if installationID == "" || playerID == "" {
		return fmt.Errorf("%w: installation_id and player_id are required", notification.ErrInvalid)
	}
	credentialHash := sha256.Sum256([]byte(credential))
	return service.store.DeletePlayerFollow(ctx, installationID, credentialHash[:], playerID)
}

func (service *Notifications) ListPlayerFollows(
	ctx context.Context,
	installationID, credential string,
	filter notification.PlayerFollowFilter,
) ([]notification.PlayerFollow, error) {
	installationID = strings.TrimSpace(installationID)
	if installationID == "" {
		return nil, fmt.Errorf("%w: installation_id is required", notification.ErrInvalid)
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		return nil, fmt.Errorf("%w: limit must be between 1 and 100", notification.ErrInvalid)
	}
	filter.BeforePlayerID = strings.TrimSpace(filter.BeforePlayerID)
	if (filter.BeforeFollowedAt == nil) != (filter.BeforePlayerID == "") {
		return nil, fmt.Errorf("%w: follow cursor requires followed_at and player_id", notification.ErrInvalid)
	}
	credentialHash := sha256.Sum256([]byte(credential))
	return service.store.ListPlayerFollows(ctx, installationID, credentialHash[:], filter)
}

func (service *Notifications) GetNotificationPreferences(
	ctx context.Context,
	installationID, credential string,
) (notification.NotificationPreferences, error) {
	installationID = strings.TrimSpace(installationID)
	if installationID == "" {
		return notification.NotificationPreferences{}, fmt.Errorf("%w: installation_id is required", notification.ErrInvalid)
	}
	credentialHash := sha256.Sum256([]byte(credential))
	return service.store.GetNotificationPreferences(ctx, installationID, credentialHash[:])
}

func (service *Notifications) SetNotificationPreferences(
	ctx context.Context,
	installationID, credential string,
	command notification.SetNotificationPreferences,
) (notification.NotificationPreferences, error) {
	installationID = strings.TrimSpace(installationID)
	if installationID == "" {
		return notification.NotificationPreferences{}, fmt.Errorf("%w: installation_id is required", notification.ErrInvalid)
	}
	credentialHash := sha256.Sum256([]byte(credential))
	return service.store.SetNotificationPreferences(ctx, installationID, credentialHash[:], command)
}
