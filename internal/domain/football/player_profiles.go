package football

import (
	"encoding/json"
	"time"
)

type PlayerMembership struct {
	ID           string       `json:"id"`
	Team         TeamSummary  `json:"team"`
	TeamType     string       `json:"team_type"`
	Role         string       `json:"role"`
	ShirtNumber  *int16       `json:"shirt_number,omitempty"`
	StartsOn     *time.Time   `json:"starts_on,omitempty"`
	EndsOn       *time.Time   `json:"ends_on,omitempty"`
	IsLoan       bool         `json:"is_loan"`
	ParentTeam   *TeamSummary `json:"parent_team,omitempty"`
	TransferType string       `json:"transfer_type"`
	IsCurrent    bool         `json:"is_current"`
}

type PlayerMemberships struct {
	PlayerID string             `json:"player_id"`
	Data     []PlayerMembership `json:"data"`
}

type PlayerPerformanceTotals struct {
	Appearances     int      `json:"appearances"`
	Starts          int      `json:"starts"`
	MinutesPlayed   int      `json:"minutes_played"`
	Goals           int      `json:"goals"`
	Assists         int      `json:"assists"`
	Shots           int      `json:"shots"`
	ShotsOnTarget   int      `json:"shots_on_target"`
	Passes          int      `json:"passes"`
	PassesCompleted *int     `json:"passes_completed,omitempty"`
	PassAccuracy    *float64 `json:"pass_accuracy,omitempty"`
	KeyPasses       *int     `json:"key_passes,omitempty"`
	Tackles         int      `json:"tackles"`
	Interceptions   *int     `json:"interceptions,omitempty"`
	Clearances      *int     `json:"clearances,omitempty"`
	Blocks          *int     `json:"blocks,omitempty"`
	Duels           *int     `json:"duels,omitempty"`
	DuelsWon        *int     `json:"duels_won,omitempty"`
	Saves           int      `json:"saves"`
	YellowCards     int      `json:"yellow_cards"`
	RedCards        int      `json:"red_cards"`
	ExpectedGoals   *float64 `json:"expected_goals,omitempty"`
	ExpectedAssists *float64 `json:"expected_assists,omitempty"`
	AverageRating   *float64 `json:"average_rating,omitempty"`
}

type PlayerStatisticsCoverage struct {
	Matches         int `json:"matches"`
	RatedMatches    int `json:"rated_matches"`
	AdvancedMatches int `json:"advanced_matches"`
}

type PlayerTeamStatistics struct {
	Team       TeamSummary              `json:"team"`
	Statistics PlayerPerformanceTotals  `json:"statistics"`
	Coverage   PlayerStatisticsCoverage `json:"coverage"`
}

type PlayerSeasonStatistics struct {
	PlayerID   string                   `json:"player_id"`
	Season     Season                   `json:"season"`
	League     League                   `json:"league"`
	Statistics PlayerPerformanceTotals  `json:"statistics"`
	ByTeam     []PlayerTeamStatistics   `json:"by_team"`
	Coverage   PlayerStatisticsCoverage `json:"coverage"`
	UpdatedAt  *time.Time               `json:"updated_at,omitempty"`
}

type PlayerSubstitution struct {
	EnteredAt           *SubstitutionDetail `json:"entered_at,omitempty"`
	LeftAt              *SubstitutionDetail `json:"left_at,omitempty"`
	ReplacedPlayerID    *string             `json:"replaced_player_id,omitempty"`
	ReplacementPlayerID *string             `json:"replacement_player_id,omitempty"`
}

type PlayerMatchHistoryItem struct {
	Fixture       Match              `json:"fixture"`
	Team          TeamSummary        `json:"team"`
	Opponent      TeamSummary        `json:"opponent"`
	VenueSide     string             `json:"venue_side"`
	Result        *string            `json:"result,omitempty"`
	Started       bool               `json:"started"`
	Minutes       int16              `json:"minutes"`
	Goals         int16              `json:"goals"`
	Assists       int16              `json:"assists"`
	Shots         int16              `json:"shots"`
	ShotsOnTarget int16              `json:"shots_on_target"`
	Passes        int16              `json:"passes"`
	YellowCards   int16              `json:"yellow_cards"`
	RedCards      int16              `json:"red_cards"`
	Rating        *float64           `json:"rating,omitempty"`
	Substitution  PlayerSubstitution `json:"substitution"`
}

type PlayerMatchFilter struct {
	SeasonID      string
	LeagueID      string
	Limit         int
	BeforeKickoff *time.Time
	BeforeMatchID string
}

type PlayerCareerSeason struct {
	Season     Season                   `json:"season"`
	League     League                   `json:"league"`
	Statistics PlayerPerformanceTotals  `json:"statistics"`
	Coverage   PlayerStatisticsCoverage `json:"coverage"`
}

type PlayerCareerSpell struct {
	Membership PlayerMembership     `json:"membership"`
	Seasons    []PlayerCareerSeason `json:"seasons"`
}

type PlayerCareer struct {
	PlayerID string              `json:"player_id"`
	Spells   []PlayerCareerSpell `json:"spells"`
}

type PlayerDiscoveryFilter struct {
	Query     string
	LeagueID  string
	SeasonID  string
	TeamID    string
	Position  string
	Limit     int
	AfterName string
	AfterID   string
}

type PlayerDiscoveryResult struct {
	Player             Player                  `json:"player"`
	CurrentMemberships []PlayerMembership      `json:"current_memberships"`
	SeasonStatistics   *PlayerSeasonStatistics `json:"season_statistics,omitempty"`
}

type PlayerComparisonFilter struct {
	PlayerIDs []string
	SeasonID  string
	LeagueID  string
}

type PlayerComparisonEntry struct {
	Player     Player                   `json:"player"`
	Statistics PlayerPerformanceTotals  `json:"statistics"`
	Coverage   PlayerStatisticsCoverage `json:"coverage"`
}

type PlayerComparison struct {
	Season     Season                  `json:"season"`
	League     League                  `json:"league"`
	Metrics    []string                `json:"metrics"`
	Players    []PlayerComparisonEntry `json:"players"`
	Compatible bool                    `json:"compatible"`
	Warnings   []string                `json:"warnings"`
}

type TraitMetric struct {
	Key        string   `json:"key"`
	Label      string   `json:"label"`
	Category   string   `json:"category"`
	RawValue   *float64 `json:"raw_value,omitempty"`
	Per90Value *float64 `json:"per_90_value,omitempty"`
	Percentile float64  `json:"percentile"`
	Unit       *string  `json:"unit,omitempty"`
	Direction  string   `json:"direction"`
}

type PlayerTraits struct {
	ID             string          `json:"id"`
	PlayerID       string          `json:"player_id"`
	TeamID         *string         `json:"team_id,omitempty"`
	LeagueID       string          `json:"league_id"`
	SeasonID       string          `json:"season_id"`
	Source         string          `json:"source"`
	ExternalID     string          `json:"external_id"`
	PositionGroup  string          `json:"position_group"`
	MinimumMinutes int             `json:"minimum_minutes"`
	CohortSize     int             `json:"cohort_size"`
	PlayerMinutes  int             `json:"player_minutes"`
	ObservedAt     time.Time       `json:"observed_at"`
	Metrics        []TraitMetric   `json:"metrics"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type UpsertPlayerTraits struct {
	TeamID         *string         `json:"team_id,omitempty"`
	LeagueID       string          `json:"league_id"`
	SeasonID       string          `json:"season_id"`
	PositionGroup  string          `json:"position_group"`
	MinimumMinutes int             `json:"minimum_minutes"`
	CohortSize     int             `json:"cohort_size"`
	PlayerMinutes  int             `json:"player_minutes"`
	ObservedAt     time.Time       `json:"observed_at"`
	Metrics        []TraitMetric   `json:"metrics"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
}

type PlayerAnalyticsFilter struct {
	SeasonID       string
	LeagueID       string
	MatchID        string
	Source         string
	Limit          int
	BeforeMinute   *int16
	BeforeSequence *int
}

type CoordinateSystem struct {
	XMin        int    `json:"x_min"`
	XMax        int    `json:"x_max"`
	YMin        int    `json:"y_min"`
	YMax        int    `json:"y_max"`
	Origin      string `json:"origin"`
	Orientation string `json:"orientation"`
}

type HeatmapPoint struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Intensity float64 `json:"intensity"`
	Touches   int     `json:"touches"`
}

type PlayerHeatmap struct {
	PlayerID         string           `json:"player_id"`
	CoordinateSystem CoordinateSystem `json:"coordinate_system"`
	Data             []HeatmapPoint   `json:"data"`
}

type TouchPointInput struct {
	Sequence       int     `json:"sequence"`
	Minute         *int16  `json:"minute,omitempty"`
	StoppageMinute *int16  `json:"stoppage_minute,omitempty"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Intensity      float64 `json:"intensity"`
	TouchType      *string `json:"touch_type,omitempty"`
}

type PlayerShot struct {
	ID             string  `json:"id"`
	MatchID        string  `json:"match_id"`
	Sequence       int     `json:"sequence"`
	Minute         *int16  `json:"minute,omitempty"`
	StoppageMinute *int16  `json:"stoppage_minute,omitempty"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	ExpectedGoals  float64 `json:"expected_goals"`
	Outcome        string  `json:"outcome"`
	BodyPart       string  `json:"body_part"`
	ShotType       *string `json:"shot_type,omitempty"`
}

type ShotInput struct {
	Sequence       int     `json:"sequence"`
	MatchEventID   *string `json:"match_event_id,omitempty"`
	Minute         *int16  `json:"minute,omitempty"`
	StoppageMinute *int16  `json:"stoppage_minute,omitempty"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	ExpectedGoals  float64 `json:"expected_goals"`
	Outcome        string  `json:"outcome"`
	BodyPart       string  `json:"body_part"`
	ShotType       *string `json:"shot_type,omitempty"`
}

type UpsertPlayerSpatial struct {
	TeamID      string            `json:"team_id"`
	Orientation string            `json:"orientation"`
	ObservedAt  time.Time         `json:"observed_at"`
	Touches     []TouchPointInput `json:"touches"`
	Shots       []ShotInput       `json:"shots"`
	Metadata    json.RawMessage   `json:"metadata,omitempty"`
}

type PlayerValuation struct {
	ID            string          `json:"id"`
	PlayerID      string          `json:"player_id"`
	TeamID        *string         `json:"team_id,omitempty"`
	AmountMinor   int64           `json:"amount_minor"`
	Currency      string          `json:"currency"`
	ValuationDate time.Time       `json:"valuation_date"`
	Source        string          `json:"source"`
	ExternalID    string          `json:"external_id"`
	ObservedAt    time.Time       `json:"observed_at"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type UpsertPlayerValuation struct {
	TeamID        *string         `json:"team_id,omitempty"`
	AmountMinor   int64           `json:"amount_minor"`
	Currency      string          `json:"currency"`
	ValuationDate time.Time       `json:"valuation_date"`
	ObservedAt    time.Time       `json:"observed_at"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}
