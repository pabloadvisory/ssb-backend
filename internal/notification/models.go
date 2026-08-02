package notification

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

var (
	ErrUnauthorized = errors.New("installation credentials are invalid")
	ErrInvalid      = errors.New("notification data is invalid")
	ErrLeaseLost    = errors.New("delivery lease is no longer owned")
)

type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
)

type Transport string

const (
	TransportAPNs Transport = "apns"
	TransportFCM  Transport = "fcm"
)

type EndpointKind string

const (
	EndpointStandard     EndpointKind = "standard"
	EndpointLiveActivity EndpointKind = "live_activity"
	EndpointPushToStart  EndpointKind = "push_to_start"
)

type DeliveryKind string

const (
	DeliveryMatchUpdate        DeliveryKind = "match_update"
	DeliveryMatchFinished      DeliveryKind = "match_finished"
	DeliveryLiveActivityStart  DeliveryKind = "live_activity_start"
	DeliveryLiveActivityUpdate DeliveryKind = "live_activity_update"
	DeliveryLiveActivityEnd    DeliveryKind = "live_activity_end"
)

type Priority string

const (
	PriorityNormal Priority = "normal"
	PriorityHigh   Priority = "high"
)

type Installation struct {
	ID         string    `json:"id"`
	Platform   Platform  `json:"platform"`
	AppID      string    `json:"app_id"`
	Locale     *string   `json:"locale,omitempty"`
	Timezone   *string   `json:"timezone,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Credential string    `json:"credential,omitempty"`
}

type CreateInstallation struct {
	Platform Platform `json:"platform"`
	AppID    string   `json:"app_id"`
	Locale   *string  `json:"locale,omitempty"`
	Timezone *string  `json:"timezone,omitempty"`
}

type Endpoint struct {
	ID                     string          `json:"id"`
	InstallationID         string          `json:"installation_id"`
	Transport              Transport       `json:"transport"`
	Kind                   EndpointKind    `json:"kind"`
	Environment            string          `json:"environment"`
	MatchID                *string         `json:"match_id,omitempty"`
	ActivityID             *string         `json:"activity_id,omitempty"`
	FrequentUpdatesEnabled bool            `json:"frequent_updates_enabled"`
	Metadata               json.RawMessage `json:"metadata,omitempty"`
	RegisteredAt           time.Time       `json:"registered_at"`
}

type RegisterEndpoint struct {
	Transport              Transport       `json:"transport"`
	Token                  string          `json:"token"`
	Environment            string          `json:"environment"`
	MatchID                *string         `json:"match_id,omitempty"`
	ActivityID             *string         `json:"activity_id,omitempty"`
	FrequentUpdatesEnabled bool            `json:"frequent_updates_enabled"`
	Metadata               json.RawMessage `json:"metadata,omitempty"`
}

type Subscription struct {
	InstallationID       string    `json:"installation_id"`
	MatchID              string    `json:"match_id"`
	NotificationsEnabled bool      `json:"notifications_enabled"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type PlayerFollow struct {
	InstallationID       string          `json:"installation_id"`
	Player               football.Player `json:"player"`
	NotificationsEnabled bool            `json:"notifications_enabled"`
	FollowedAt           time.Time       `json:"followed_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}

type SetPlayerFollow struct {
	NotificationsEnabled bool `json:"notifications_enabled"`
}

type PlayerFollowFilter struct {
	Limit            int
	BeforeFollowedAt *time.Time
	BeforePlayerID   string
}

type NotificationPreferences struct {
	InstallationID              string    `json:"installation_id"`
	MatchUpdatesEnabled         bool      `json:"match_updates_enabled"`
	MatchFinishedEnabled        bool      `json:"match_finished_enabled"`
	FollowedPlayerEventsEnabled bool      `json:"followed_player_events_enabled"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

type SetNotificationPreferences struct {
	MatchUpdatesEnabled         bool `json:"match_updates_enabled"`
	MatchFinishedEnabled        bool `json:"match_finished_enabled"`
	FollowedPlayerEventsEnabled bool `json:"followed_player_events_enabled"`
}

type Delivery struct {
	ID                     string
	LockToken              string
	EndpointID             string
	Transport              Transport
	Kind                   DeliveryKind
	Token                  string
	Environment            string
	AppID                  string
	MatchID                *string
	FrequentUpdatesEnabled bool
	Payload                json.RawMessage
	CollapseKey            *string
	Priority               Priority
	Attempts               int
}

type MatchUpdate struct {
	MatchID       string                `json:"match_id"`
	Version       int64                 `json:"version"`
	Status        football.MatchStatus  `json:"status"`
	HomeTeamID    string                `json:"home_team_id"`
	HomeTeamName  string                `json:"home_team_name"`
	AwayTeamID    string                `json:"away_team_id"`
	AwayTeamName  string                `json:"away_team_name"`
	HomeScore     *int16                `json:"home_score,omitempty"`
	AwayScore     *int16                `json:"away_score,omitempty"`
	ElapsedMinute *int16                `json:"elapsed_minute,omitempty"`
	Period        *football.MatchPeriod `json:"period,omitempty"`
	KickoffAtUnix int64                 `json:"kickoff_at_unix"`
}

type Store interface {
	CreateInstallation(context.Context, CreateInstallation, []byte) (Installation, error)
	RegisterEndpoint(context.Context, string, []byte, EndpointKind, RegisterEndpoint, []byte) (Endpoint, error)
	SetMatchSubscription(context.Context, string, []byte, string, bool) (Subscription, error)
	SetPlayerFollow(context.Context, string, []byte, string, SetPlayerFollow) (PlayerFollow, error)
	DeletePlayerFollow(context.Context, string, []byte, string) error
	ListPlayerFollows(context.Context, string, []byte, PlayerFollowFilter) ([]PlayerFollow, error)
	GetNotificationPreferences(context.Context, string, []byte) (NotificationPreferences, error)
	SetNotificationPreferences(context.Context, string, []byte, SetNotificationPreferences) (NotificationPreferences, error)
	ClaimDeliveries(context.Context, int, time.Duration) ([]Delivery, error)
	CompleteDelivery(context.Context, string, string, string) error
	RetryDelivery(context.Context, string, string, time.Time, string, bool) error
	InvalidateEndpoint(context.Context, string, string) error
}
