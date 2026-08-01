package football

import "encoding/json"

type MatchTeamInfoInput struct {
	TeamID    string          `json:"team_id"`
	Formation *string         `json:"formation,omitempty"`
	CoachID   *string         `json:"coach_id,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type LineupInput struct {
	TeamID       string          `json:"team_id"`
	PersonID     string          `json:"person_id"`
	Position     *string         `json:"position,omitempty"`
	GridPosition *string         `json:"grid_position,omitempty"`
	ShirtNumber  *int16          `json:"shirt_number,omitempty"`
	IsStarter    bool            `json:"is_starter"`
	IsCaptain    bool            `json:"is_captain"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

type TeamStatisticsInput struct {
	TeamID string `json:"team_id"`
	MatchTeamTotals
}

type PlayerStatisticsInput struct {
	TeamID        string          `json:"team_id"`
	PersonID      string          `json:"person_id"`
	Started       bool            `json:"started"`
	MinutesPlayed int16           `json:"minutes_played"`
	Goals         int16           `json:"goals"`
	Assists       int16           `json:"assists"`
	Shots         int16           `json:"shots"`
	ShotsOnTarget int16           `json:"shots_on_target"`
	Passes        int16           `json:"passes"`
	Tackles       int16           `json:"tackles"`
	Saves         int16           `json:"saves"`
	YellowCards   int16           `json:"yellow_cards"`
	RedCards      int16           `json:"red_cards"`
	Rating        *float64        `json:"rating,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type OfficialInput struct {
	PersonID string            `json:"person_id"`
	Role     MatchOfficialRole `json:"role"`
	Metadata json.RawMessage   `json:"metadata,omitempty"`
}

type MatchCoverageUpdate struct {
	TeamInfo         *[]MatchTeamInfoInput    `json:"team_info,omitempty"`
	Lineups          *[]LineupInput           `json:"lineups,omitempty"`
	TeamStatistics   *[]TeamStatisticsInput   `json:"team_statistics,omitempty"`
	PlayerStatistics *[]PlayerStatisticsInput `json:"player_statistics,omitempty"`
	Officials        *[]OfficialInput         `json:"officials,omitempty"`
}

type StandingInput struct {
	TeamID       string          `json:"team_id"`
	GroupName    string          `json:"group_name"`
	Position     int16           `json:"position"`
	Played       int16           `json:"played"`
	Won          int16           `json:"won"`
	Drawn        int16           `json:"drawn"`
	Lost         int16           `json:"lost"`
	GoalsFor     int16           `json:"goals_for"`
	GoalsAgainst int16           `json:"goals_against"`
	Points       int16           `json:"points"`
	Form         *string         `json:"form,omitempty"`
	Zone         *string         `json:"zone,omitempty"`
	Description  *string         `json:"description,omitempty"`
	HomeRecord   *StandingRecord `json:"home_record,omitempty"`
	AwayRecord   *StandingRecord `json:"away_record,omitempty"`
}

type StandingsUpdate struct {
	Data []StandingInput `json:"data"`
}
