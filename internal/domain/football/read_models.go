package football

import (
	"encoding/json"
	"time"
)

type TeamSummary struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	ShortName      *string `json:"short_name,omitempty"`
	Code           *string `json:"code,omitempty"`
	LogoURL        *string `json:"logo_url,omitempty"`
	PrimaryColor   *string `json:"primary_color,omitempty"`
	SecondaryColor *string `json:"secondary_color,omitempty"`
}

type SubstitutionDetail struct {
	Period              *MatchPeriod `json:"period,omitempty"`
	Minute              *int16       `json:"minute,omitempty"`
	StoppageMinute      *int16       `json:"stoppage_minute,omitempty"`
	ReplacementPlayerID *string      `json:"replacement_player_id,omitempty"`
	ReplacedPlayerID    *string      `json:"replaced_player_id,omitempty"`
}

type LineupPlayer struct {
	Player             Player              `json:"player"`
	ShirtNumber        *int16              `json:"shirt_number,omitempty"`
	LineupPosition     *string             `json:"lineup_position,omitempty"`
	GridPosition       *string             `json:"grid_position,omitempty"`
	IsCaptain          bool                `json:"is_captain"`
	SubstitutionStatus string              `json:"substitution_status"`
	SubstitutedIn      *SubstitutionDetail `json:"substituted_in,omitempty"`
	SubstitutedOut     *SubstitutionDetail `json:"substituted_out,omitempty"`
}

type TeamLineup struct {
	Team        TeamSummary    `json:"team"`
	Formation   *string        `json:"formation,omitempty"`
	Coach       *Coach         `json:"coach,omitempty"`
	Starters    []LineupPlayer `json:"starters"`
	Substitutes []LineupPlayer `json:"substitutes"`
}

type MatchLineups struct {
	MatchID string     `json:"match_id"`
	Home    TeamLineup `json:"home"`
	Away    TeamLineup `json:"away"`
}

type MatchTeamTotals struct {
	Possession      *float64        `json:"possession,omitempty"`
	Shots           *int16          `json:"shots,omitempty"`
	ShotsOnTarget   *int16          `json:"shots_on_target,omitempty"`
	ShotsOffTarget  *int16          `json:"shots_off_target,omitempty"`
	BlockedShots    *int16          `json:"blocked_shots,omitempty"`
	ShotsInsideBox  *int16          `json:"shots_inside_box,omitempty"`
	ShotsOutsideBox *int16          `json:"shots_outside_box,omitempty"`
	Corners         *int16          `json:"corners,omitempty"`
	Passes          *int16          `json:"passes,omitempty"`
	PassesCompleted *int16          `json:"passes_completed,omitempty"`
	PassAccuracy    *float64        `json:"pass_accuracy,omitempty"`
	Fouls           *int16          `json:"fouls,omitempty"`
	Offsides        *int16          `json:"offsides,omitempty"`
	YellowCards     *int16          `json:"yellow_cards,omitempty"`
	RedCards        *int16          `json:"red_cards,omitempty"`
	Saves           *int16          `json:"saves,omitempty"`
	Tackles         *int16          `json:"tackles,omitempty"`
	Interceptions   *int16          `json:"interceptions,omitempty"`
	Clearances      *int16          `json:"clearances,omitempty"`
	ExpectedGoals   *float64        `json:"expected_goals,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type PlayerStatistics struct {
	Player          Player   `json:"player"`
	Started         bool     `json:"started"`
	MinutesPlayed   int16    `json:"minutes_played"`
	Goals           int16    `json:"goals"`
	Assists         int16    `json:"assists"`
	Shots           int16    `json:"shots"`
	ShotsOnTarget   int16    `json:"shots_on_target"`
	Passes          int16    `json:"passes"`
	PassesCompleted *int16   `json:"passes_completed,omitempty"`
	KeyPasses       *int16   `json:"key_passes,omitempty"`
	Tackles         int16    `json:"tackles"`
	Interceptions   *int16   `json:"interceptions,omitempty"`
	Clearances      *int16   `json:"clearances,omitempty"`
	Blocks          *int16   `json:"blocks,omitempty"`
	Duels           *int16   `json:"duels,omitempty"`
	DuelsWon        *int16   `json:"duels_won,omitempty"`
	Saves           int16    `json:"saves"`
	YellowCards     int16    `json:"yellow_cards"`
	RedCards        int16    `json:"red_cards"`
	Rating          *float64 `json:"rating,omitempty"`
	ExpectedGoals   *float64 `json:"expected_goals,omitempty"`
	ExpectedAssists *float64 `json:"expected_assists,omitempty"`
}

type TeamMatchStatistics struct {
	Team    TeamSummary        `json:"team"`
	Totals  *MatchTeamTotals   `json:"totals"`
	Players []PlayerStatistics `json:"players"`
}

type MatchStatistics struct {
	MatchID string              `json:"match_id"`
	Home    TeamMatchStatistics `json:"home"`
	Away    TeamMatchStatistics `json:"away"`
}

type StandingRecord struct {
	Played int16 `json:"played"`
	Won    int16 `json:"won"`
	Drawn  int16 `json:"drawn"`
	Lost   int16 `json:"lost"`
}

type StandingEntry struct {
	GroupName      string          `json:"group_name"`
	Position       int16           `json:"position"`
	Team           TeamSummary     `json:"team"`
	Played         int16           `json:"played"`
	Won            int16           `json:"won"`
	Drawn          int16           `json:"drawn"`
	Lost           int16           `json:"lost"`
	GoalsFor       int16           `json:"goals_for"`
	GoalsAgainst   int16           `json:"goals_against"`
	GoalDifference int16           `json:"goal_difference"`
	Points         int16           `json:"points"`
	Form           *string         `json:"form,omitempty"`
	Zone           *string         `json:"zone,omitempty"`
	Description    *string         `json:"description,omitempty"`
	HomeRecord     *StandingRecord `json:"home_record,omitempty"`
	AwayRecord     *StandingRecord `json:"away_record,omitempty"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type SeasonStandings struct {
	SeasonID string          `json:"season_id"`
	Data     []StandingEntry `json:"data"`
}

type MatchOfficialDetail struct {
	Role   MatchOfficialRole `json:"role"`
	Person Person            `json:"person"`
}

type MatchOfficials struct {
	MatchID string                `json:"match_id"`
	Data    []MatchOfficialDetail `json:"data"`
}

type SearchEntityType string

const (
	SearchLeague  SearchEntityType = "league"
	SearchTeam    SearchEntityType = "team"
	SearchPlayer  SearchEntityType = "player"
	SearchCoach   SearchEntityType = "coach"
	SearchFixture SearchEntityType = "fixture"
)

type SearchFilter struct {
	Query string
	Types []SearchEntityType
	Limit int
}

type SearchResult struct {
	EntityType   SearchEntityType `json:"entity_type"`
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	ImageURL     *string          `json:"image_url,omitempty"`
	CountryCode  *string          `json:"country_code,omitempty"`
	TeamCode     *string          `json:"team_code,omitempty"`
	Position     *string          `json:"position,omitempty"`
	LeagueID     *string          `json:"league_id,omitempty"`
	SeasonID     *string          `json:"season_id,omitempty"`
	KickoffAt    *time.Time       `json:"kickoff_at,omitempty"`
	MatchStatus  *MatchStatus     `json:"match_status,omitempty"`
	HomeTeamID   *string          `json:"home_team_id,omitempty"`
	HomeTeamName *string          `json:"home_team_name,omitempty"`
	AwayTeamID   *string          `json:"away_team_id,omitempty"`
	AwayTeamName *string          `json:"away_team_name,omitempty"`
}

type LeagueSeasons struct {
	LeagueID string   `json:"league_id"`
	Data     []Season `json:"data"`
}

type HeadToHeadFilter struct {
	TeamAID string
	TeamBID string
	Limit   int
}

type HeadToHeadSummary struct {
	TeamAWins         int `json:"team_a_wins"`
	Draws             int `json:"draws"`
	TeamBWins         int `json:"team_b_wins"`
	MatchesConsidered int `json:"matches_considered"`
}

type HeadToHead struct {
	TeamAID  string            `json:"team_a_id"`
	TeamBID  string            `json:"team_b_id"`
	Summary  HeadToHeadSummary `json:"summary"`
	Meetings []Match           `json:"meetings"`
}
