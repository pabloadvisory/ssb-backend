package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pabloadvisory/ssb-backend/internal/notification"
)

type Notifications struct {
	store notification.Store
}

func NewNotifications(store notification.Store) *Notifications {
	return &Notifications{store: store}
}

func (service *Notifications) CreateInstallation(ctx context.Context, command notification.CreateInstallation) (notification.Installation, error) {
	command.AppID = strings.TrimSpace(command.AppID)
	if command.AppID == "" || len(command.AppID) > 255 {
		return notification.Installation{}, fmt.Errorf("%w: app_id is required", notification.ErrInvalid)
	}
	if command.Platform != notification.PlatformIOS && command.Platform != notification.PlatformAndroid {
		return notification.Installation{}, fmt.Errorf("%w: platform must be ios or android", notification.ErrInvalid)
	}
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return notification.Installation{}, fmt.Errorf("generate installation credential: %w", err)
	}
	credential := base64.RawURLEncoding.EncodeToString(secretBytes)
	hash := sha256.Sum256([]byte(credential))
	installation, err := service.store.CreateInstallation(ctx, command, hash[:])
	if err != nil {
		return notification.Installation{}, err
	}
	installation.Credential = credential
	return installation, nil
}

func (service *Notifications) RegisterEndpoint(ctx context.Context, installationID, credential string, kind notification.EndpointKind, command notification.RegisterEndpoint) (notification.Endpoint, error) {
	command.Token = strings.TrimSpace(command.Token)
	if command.Token == "" || len(command.Token) > 4096 {
		return notification.Endpoint{}, fmt.Errorf("%w: token is required", notification.ErrInvalid)
	}
	if kind != notification.EndpointStandard && kind != notification.EndpointLiveActivity && kind != notification.EndpointPushToStart {
		return notification.Endpoint{}, fmt.Errorf("%w: endpoint kind is invalid", notification.ErrInvalid)
	}
	if command.Transport != notification.TransportAPNs && command.Transport != notification.TransportFCM {
		return notification.Endpoint{}, fmt.Errorf("%w: transport must be apns or fcm", notification.ErrInvalid)
	}
	if command.Environment == "" {
		command.Environment = "production"
	}
	if command.Environment != "production" && command.Environment != "sandbox" {
		return notification.Endpoint{}, fmt.Errorf("%w: environment must be production or sandbox", notification.ErrInvalid)
	}
	if kind == notification.EndpointLiveActivity && (command.Transport != notification.TransportAPNs || command.MatchID == nil || command.ActivityID == nil) {
		return notification.Endpoint{}, fmt.Errorf("%w: live_activity requires APNs, match_id, and activity_id", notification.ErrInvalid)
	}
	if kind == notification.EndpointPushToStart && (command.Transport != notification.TransportAPNs || command.MatchID != nil) {
		return notification.Endpoint{}, fmt.Errorf("%w: push_to_start requires APNs and no match_id", notification.ErrInvalid)
	}
	if len(command.Metadata) == 0 {
		command.Metadata = json.RawMessage(`{}`)
	}
	if !json.Valid(command.Metadata) {
		return notification.Endpoint{}, fmt.Errorf("%w: metadata must be valid JSON", notification.ErrInvalid)
	}
	credentialHash := sha256.Sum256([]byte(credential))
	tokenHash := sha256.Sum256([]byte(command.Token))
	return service.store.RegisterEndpoint(ctx, installationID, credentialHash[:], kind, command, tokenHash[:])
}

func (service *Notifications) SetMatchSubscription(ctx context.Context, installationID, credential, matchID string, enabled bool) (notification.Subscription, error) {
	credentialHash := sha256.Sum256([]byte(credential))
	return service.store.SetMatchSubscription(ctx, installationID, credentialHash[:], matchID, enabled)
}
