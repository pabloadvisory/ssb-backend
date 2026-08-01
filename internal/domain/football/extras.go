package football

import (
	"encoding/json"
	"time"
)

type OddsSelection struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Line        *float64        `json:"line,omitempty"`
	DecimalOdds float64         `json:"decimal_odds"`
	Result      *string         `json:"result,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type OddsMarket struct {
	Key        string          `json:"key"`
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Selections []OddsSelection `json:"selections"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type MatchOddsSnapshot struct {
	ID               string          `json:"id"`
	MatchID          string          `json:"match_id"`
	Source           string          `json:"source"`
	ExternalID       string          `json:"external_id"`
	BookmakerSlug    string          `json:"bookmaker_slug"`
	BookmakerName    string          `json:"bookmaker_name"`
	BookmakerLogoURL *string         `json:"bookmaker_logo_url,omitempty"`
	ObservedAt       time.Time       `json:"observed_at"`
	ValidUntil       *time.Time      `json:"valid_until,omitempty"`
	Markets          []OddsMarket    `json:"markets"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

type MatchOdds struct {
	MatchID string              `json:"match_id"`
	Data    []MatchOddsSnapshot `json:"data"`
}

type UpsertOddsSnapshot struct {
	BookmakerSlug       string          `json:"bookmaker_slug"`
	BookmakerName       string          `json:"bookmaker_name"`
	BookmakerLogoURL    *string         `json:"bookmaker_logo_url,omitempty"`
	BookmakerWebsiteURL *string         `json:"bookmaker_website_url,omitempty"`
	ObservedAt          time.Time       `json:"observed_at"`
	ValidUntil          *time.Time      `json:"valid_until,omitempty"`
	Markets             []OddsMarket    `json:"markets"`
	Metadata            json.RawMessage `json:"metadata,omitempty"`
}

type BroadcastKind string

const (
	BroadcastTelevision BroadcastKind = "television"
	BroadcastRadio      BroadcastKind = "radio"
	BroadcastStream     BroadcastKind = "stream"
)

type BroadcastScope string

const (
	BroadcastGlobal      BroadcastScope = "global"
	BroadcastTerritorial BroadcastScope = "territorial"
	BroadcastUnknown     BroadcastScope = "unknown"
)

type BroadcastStatus string

const (
	BroadcastScheduled   BroadcastStatus = "scheduled"
	BroadcastLive        BroadcastStatus = "live"
	BroadcastEnded       BroadcastStatus = "ended"
	BroadcastCancelled   BroadcastStatus = "cancelled"
	BroadcastUnavailable BroadcastStatus = "unavailable"
)

type MatchBroadcast struct {
	ID                   string          `json:"id"`
	MatchID              string          `json:"match_id"`
	Source               string          `json:"source"`
	ExternalID           string          `json:"external_id"`
	NetworkName          string          `json:"network_name"`
	ServiceName          *string         `json:"service_name,omitempty"`
	Kind                 BroadcastKind   `json:"kind"`
	AvailabilityScope    BroadcastScope  `json:"availability_scope"`
	Regions              []string        `json:"regions"`
	LanguageTags         []string        `json:"language_tags"`
	StartsAt             *time.Time      `json:"starts_at,omitempty"`
	EndsAt               *time.Time      `json:"ends_at,omitempty"`
	IsFree               bool            `json:"is_free"`
	RequiresSubscription bool            `json:"requires_subscription"`
	WebURL               *string         `json:"web_url,omitempty"`
	DeepLinkURL          *string         `json:"deep_link_url,omitempty"`
	Status               BroadcastStatus `json:"status"`
	ObservedAt           time.Time       `json:"observed_at"`
	Metadata             json.RawMessage `json:"metadata,omitempty"`
}

type MatchBroadcasts struct {
	MatchID string           `json:"match_id"`
	Data    []MatchBroadcast `json:"data"`
}

type UpsertMatchBroadcast struct {
	NetworkName          string          `json:"network_name"`
	ServiceName          *string         `json:"service_name,omitempty"`
	Kind                 BroadcastKind   `json:"kind"`
	AvailabilityScope    BroadcastScope  `json:"availability_scope"`
	Regions              []string        `json:"regions"`
	LanguageTags         []string        `json:"language_tags"`
	StartsAt             *time.Time      `json:"starts_at,omitempty"`
	EndsAt               *time.Time      `json:"ends_at,omitempty"`
	IsFree               bool            `json:"is_free"`
	RequiresSubscription bool            `json:"requires_subscription"`
	WebURL               *string         `json:"web_url,omitempty"`
	DeepLinkURL          *string         `json:"deep_link_url,omitempty"`
	Status               BroadcastStatus `json:"status"`
	ObservedAt           time.Time       `json:"observed_at"`
	Metadata             json.RawMessage `json:"metadata,omitempty"`
}

type WeatherKind string

const (
	WeatherForecast WeatherKind = "forecast"
	WeatherObserved WeatherKind = "observed"
)

type WeatherSnapshot struct {
	ID                              string          `json:"id"`
	MatchID                         string          `json:"match_id"`
	Source                          string          `json:"source"`
	ExternalID                      string          `json:"external_id"`
	Kind                            WeatherKind     `json:"kind"`
	ValidAt                         time.Time       `json:"valid_at"`
	IssuedAt                        time.Time       `json:"issued_at"`
	ReceivedAt                      time.Time       `json:"received_at"`
	TemperatureC                    *float64        `json:"temperature_c,omitempty"`
	FeelsLikeC                      *float64        `json:"feels_like_c,omitempty"`
	HumidityPercent                 *float64        `json:"humidity_percent,omitempty"`
	PrecipitationProbabilityPercent *float64        `json:"precipitation_probability_percent,omitempty"`
	PrecipitationMM                 *float64        `json:"precipitation_mm,omitempty"`
	WindSpeedKPH                    *float64        `json:"wind_speed_kph,omitempty"`
	WindGustKPH                     *float64        `json:"wind_gust_kph,omitempty"`
	WindDirectionDegrees            *int16          `json:"wind_direction_degrees,omitempty"`
	PressureHPA                     *float64        `json:"pressure_hpa,omitempty"`
	VisibilityKM                    *float64        `json:"visibility_km,omitempty"`
	ConditionCode                   *string         `json:"condition_code,omitempty"`
	ConditionText                   *string         `json:"condition_text,omitempty"`
	IconURL                         *string         `json:"icon_url,omitempty"`
	Metadata                        json.RawMessage `json:"metadata,omitempty"`
}

type MatchWeather struct {
	MatchID     string           `json:"match_id"`
	Forecast    *WeatherSnapshot `json:"forecast,omitempty"`
	Observation *WeatherSnapshot `json:"observation,omitempty"`
}

type UpsertWeatherSnapshot struct {
	Kind                            WeatherKind     `json:"kind"`
	ValidAt                         time.Time       `json:"valid_at"`
	IssuedAt                        time.Time       `json:"issued_at"`
	TemperatureC                    *float64        `json:"temperature_c,omitempty"`
	FeelsLikeC                      *float64        `json:"feels_like_c,omitempty"`
	HumidityPercent                 *float64        `json:"humidity_percent,omitempty"`
	PrecipitationProbabilityPercent *float64        `json:"precipitation_probability_percent,omitempty"`
	PrecipitationMM                 *float64        `json:"precipitation_mm,omitempty"`
	WindSpeedKPH                    *float64        `json:"wind_speed_kph,omitempty"`
	WindGustKPH                     *float64        `json:"wind_gust_kph,omitempty"`
	WindDirectionDegrees            *int16          `json:"wind_direction_degrees,omitempty"`
	PressureHPA                     *float64        `json:"pressure_hpa,omitempty"`
	VisibilityKM                    *float64        `json:"visibility_km,omitempty"`
	ConditionCode                   *string         `json:"condition_code,omitempty"`
	ConditionText                   *string         `json:"condition_text,omitempty"`
	IconURL                         *string         `json:"icon_url,omitempty"`
	Metadata                        json.RawMessage `json:"metadata,omitempty"`
}

type PredictionSelection string

const (
	PredictionHome PredictionSelection = "home"
	PredictionDraw PredictionSelection = "draw"
	PredictionAway PredictionSelection = "away"
)

type PredictionOption struct {
	Selection PredictionSelection `json:"selection"`
	TeamID    *string             `json:"team_id,omitempty"`
	Votes     int64               `json:"votes"`
	Percent   float64             `json:"percent"`
}

type MatchPrediction struct {
	MatchID     string               `json:"match_id"`
	ClosesAt    time.Time            `json:"closes_at"`
	IsOpen      bool                 `json:"is_open"`
	TotalVotes  int64                `json:"total_votes"`
	Options     []PredictionOption   `json:"options"`
	MySelection *PredictionSelection `json:"my_selection,omitempty"`
}
