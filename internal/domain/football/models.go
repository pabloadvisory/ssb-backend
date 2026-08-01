package football

import (
	"encoding/json"
	"time"
)

type League struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	Type            string    `json:"type"`
	Gender          string    `json:"gender"`
	CountryCode     *string   `json:"country_code,omitempty"`
	LogoURL         *string   `json:"logo_url,omitempty"`
	CurrentSeasonID *string   `json:"current_season_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Season struct {
	ID        string    `json:"id"`
	LeagueID  string    `json:"league_id"`
	Name      string    `json:"name"`
	StartsOn  time.Time `json:"starts_on"`
	EndsOn    time.Time `json:"ends_on"`
	IsCurrent bool      `json:"is_current"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Team struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	ShortName      *string   `json:"short_name,omitempty"`
	Code           *string   `json:"code,omitempty"`
	CountryCode    *string   `json:"country_code,omitempty"`
	FoundedYear    *int      `json:"founded_year,omitempty"`
	LogoURL        *string   `json:"logo_url,omitempty"`
	PrimaryColor   *string   `json:"primary_color,omitempty"`
	SecondaryColor *string   `json:"secondary_color,omitempty"`
	Venue          *Venue    `json:"venue,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Venue struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	City        *string   `json:"city,omitempty"`
	CountryCode *string   `json:"country_code,omitempty"`
	CountryName *string   `json:"country_name,omitempty"`
	Address     *string   `json:"address,omitempty"`
	Latitude    *float64  `json:"latitude,omitempty"`
	Longitude   *float64  `json:"longitude,omitempty"`
	Capacity    *int      `json:"capacity,omitempty"`
	Surface     *string   `json:"surface,omitempty"`
	ImageURL    *string   `json:"image_url,omitempty"`
	Timezone    *string   `json:"timezone,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Person struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	FirstName   *string    `json:"first_name,omitempty"`
	LastName    *string    `json:"last_name,omitempty"`
	BirthDate   *time.Time `json:"birth_date,omitempty"`
	CountryCode *string    `json:"country_code,omitempty"`
	PhotoURL    *string    `json:"photo_url,omitempty"`
}

type Player struct {
	Person
	Position         *string `json:"position,omitempty"`
	DetailedPosition *string `json:"detailed_position,omitempty"`
	PreferredFoot    *string `json:"preferred_foot,omitempty"`
	HeightCM         *int    `json:"height_cm,omitempty"`
}

type Coach struct {
	Person
}

type SeasonTeam struct {
	SeasonID  string          `json:"season_id"`
	TeamID    string          `json:"team_id"`
	Promoted  bool            `json:"promoted"`
	Relegated bool            `json:"relegated"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type MatchTeamInfo struct {
	MatchID   string          `json:"match_id"`
	TeamID    string          `json:"team_id"`
	Formation *string         `json:"formation,omitempty"`
	CoachID   *string         `json:"coach_id,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type MatchOfficialRole string

const (
	OfficialReferee          MatchOfficialRole = "referee"
	OfficialAssistantReferee MatchOfficialRole = "assistant_referee"
	OfficialFourth           MatchOfficialRole = "fourth_official"
	OfficialVAR              MatchOfficialRole = "var"
	OfficialAssistantVAR     MatchOfficialRole = "assistant_var"
)

type MatchOfficial struct {
	MatchID  string            `json:"match_id"`
	PersonID string            `json:"person_id"`
	Role     MatchOfficialRole `json:"role"`
	Metadata json.RawMessage   `json:"metadata,omitempty"`
}

type PlayerMatchStatistics struct {
	MatchID       string          `json:"match_id"`
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

type Standing struct {
	SeasonID     string    `json:"season_id"`
	TeamID       string    `json:"team_id"`
	GroupName    string    `json:"group_name"`
	Position     int16     `json:"position"`
	Played       int16     `json:"played"`
	Won          int16     `json:"won"`
	Drawn        int16     `json:"drawn"`
	Lost         int16     `json:"lost"`
	GoalsFor     int16     `json:"goals_for"`
	GoalsAgainst int16     `json:"goals_against"`
	Points       int16     `json:"points"`
	Form         *string   `json:"form,omitempty"`
	Zone         *string   `json:"zone,omitempty"`
	Description  *string   `json:"description,omitempty"`
	HomePlayed   *int16    `json:"home_played,omitempty"`
	HomeWon      *int16    `json:"home_won,omitempty"`
	HomeDrawn    *int16    `json:"home_drawn,omitempty"`
	HomeLost     *int16    `json:"home_lost,omitempty"`
	AwayPlayed   *int16    `json:"away_played,omitempty"`
	AwayWon      *int16    `json:"away_won,omitempty"`
	AwayDrawn    *int16    `json:"away_drawn,omitempty"`
	AwayLost     *int16    `json:"away_lost,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type MatchStatus string

const (
	MatchScheduled MatchStatus = "scheduled"
	MatchPostponed MatchStatus = "postponed"
	MatchCancelled MatchStatus = "cancelled"
	MatchLive      MatchStatus = "live"
	MatchSuspended MatchStatus = "suspended"
	MatchFinished  MatchStatus = "finished"
	MatchAbandoned MatchStatus = "abandoned"
	MatchAwarded   MatchStatus = "awarded"
)

type MatchPeriod string

const (
	PeriodFirstHalf           MatchPeriod = "first_half"
	PeriodHalfTime            MatchPeriod = "half_time"
	PeriodSecondHalf          MatchPeriod = "second_half"
	PeriodExtraTimeFirstHalf  MatchPeriod = "extra_time_first_half"
	PeriodExtraTimeHalfTime   MatchPeriod = "extra_time_half_time"
	PeriodExtraTimeSecondHalf MatchPeriod = "extra_time_second_half"
	PeriodPenalties           MatchPeriod = "penalties"
	PeriodFullTime            MatchPeriod = "full_time"
)

type EventType string

const (
	EventKickoff          EventType = "kickoff"
	EventHalfTime         EventType = "half_time"
	EventSecondHalf       EventType = "second_half_started"
	EventExtraTime        EventType = "extra_time_started"
	EventExtraTimeHalf    EventType = "extra_time_half_time"
	EventPenaltiesStarted EventType = "penalties_started"
	EventFullTime         EventType = "full_time"
	EventGoal             EventType = "goal"
	EventOwnGoal          EventType = "own_goal"
	EventPenaltyGoal      EventType = "penalty_goal"
	EventPenaltyMissed    EventType = "penalty_missed"
	EventYellowCard       EventType = "yellow_card"
	EventSecondYellow     EventType = "second_yellow"
	EventRedCard          EventType = "red_card"
	EventSubstitution     EventType = "substitution"
	EventVARDecision      EventType = "var_decision"
	EventMatchSuspended   EventType = "match_suspended"
	EventMatchResumed     EventType = "match_resumed"
	EventMatchCancelled   EventType = "match_cancelled"
)

type Match struct {
	ID                 string          `json:"id"`
	LeagueID           string          `json:"league_id"`
	SeasonID           string          `json:"season_id"`
	Stage              *string         `json:"stage,omitempty"`
	Round              *string         `json:"round,omitempty"`
	RoundSort          *int            `json:"round_sort,omitempty"`
	GroupName          *string         `json:"group_name,omitempty"`
	Leg                int16           `json:"leg"`
	KickoffAt          time.Time       `json:"kickoff_at"`
	Status             MatchStatus     `json:"status"`
	Period             *MatchPeriod    `json:"period,omitempty"`
	ElapsedMinute      *int16          `json:"elapsed_minute,omitempty"`
	VenueID            *string         `json:"venue_id,omitempty"`
	HomeTeamID         string          `json:"home_team_id"`
	AwayTeamID         string          `json:"away_team_id"`
	HomeScore          *int16          `json:"home_score,omitempty"`
	AwayScore          *int16          `json:"away_score,omitempty"`
	HomeHTScore        *int16          `json:"home_half_time_score,omitempty"`
	AwayHTScore        *int16          `json:"away_half_time_score,omitempty"`
	HomeExtraTimeScore *int16          `json:"home_extra_time_score,omitempty"`
	AwayExtraTimeScore *int16          `json:"away_extra_time_score,omitempty"`
	HomePenaltyScore   *int16          `json:"home_penalty_score,omitempty"`
	AwayPenaltyScore   *int16          `json:"away_penalty_score,omitempty"`
	Attendance         *int            `json:"attendance,omitempty"`
	FirstLegMatchID    *string         `json:"first_leg_match_id,omitempty"`
	WinnerTeamID       *string         `json:"winner_team_id,omitempty"`
	Version            int64           `json:"version"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type MatchEvent struct {
	ID                string          `json:"id"`
	MatchID           string          `json:"match_id"`
	Sequence          int             `json:"sequence"`
	Period            *MatchPeriod    `json:"period,omitempty"`
	Minute            *int16          `json:"minute,omitempty"`
	StoppageMinute    *int16          `json:"stoppage_minute,omitempty"`
	Type              EventType       `json:"type"`
	TeamID            *string         `json:"team_id,omitempty"`
	PrimaryPersonID   *string         `json:"primary_person_id,omitempty"`
	SecondaryPersonID *string         `json:"secondary_person_id,omitempty"`
	Detail            *string         `json:"detail,omitempty"`
	HomeScore         *int16          `json:"home_score,omitempty"`
	AwayScore         *int16          `json:"away_score,omitempty"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	OccurredAt        *time.Time      `json:"occurred_at,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
}

type MatchSnapshot struct {
	LeagueID           string          `json:"league_id"`
	SeasonID           string          `json:"season_id"`
	Stage              *string         `json:"stage,omitempty"`
	Round              *string         `json:"round,omitempty"`
	RoundSort          *int            `json:"round_sort,omitempty"`
	GroupName          *string         `json:"group_name,omitempty"`
	Leg                int16           `json:"leg"`
	KickoffAt          time.Time       `json:"kickoff_at"`
	Status             MatchStatus     `json:"status"`
	Period             *MatchPeriod    `json:"period,omitempty"`
	ElapsedMinute      *int16          `json:"elapsed_minute,omitempty"`
	VenueID            *string         `json:"venue_id,omitempty"`
	HomeTeamID         string          `json:"home_team_id"`
	AwayTeamID         string          `json:"away_team_id"`
	HomeScore          *int16          `json:"home_score,omitempty"`
	AwayScore          *int16          `json:"away_score,omitempty"`
	HomeHTScore        *int16          `json:"home_half_time_score,omitempty"`
	AwayHTScore        *int16          `json:"away_half_time_score,omitempty"`
	HomeExtraTimeScore *int16          `json:"home_extra_time_score,omitempty"`
	AwayExtraTimeScore *int16          `json:"away_extra_time_score,omitempty"`
	HomePenaltyScore   *int16          `json:"home_penalty_score,omitempty"`
	AwayPenaltyScore   *int16          `json:"away_penalty_score,omitempty"`
	Attendance         *int            `json:"attendance,omitempty"`
	FirstLegMatchID    *string         `json:"first_leg_match_id,omitempty"`
	WinnerTeamID       *string         `json:"winner_team_id,omitempty"`
	Metadata           json.RawMessage `json:"metadata,omitempty"`
	Events             []EventSnapshot `json:"events,omitempty"`
}

type EventSnapshot struct {
	ExternalID        string          `json:"external_id"`
	Sequence          int             `json:"sequence"`
	Period            *MatchPeriod    `json:"period,omitempty"`
	Minute            *int16          `json:"minute,omitempty"`
	StoppageMinute    *int16          `json:"stoppage_minute,omitempty"`
	Type              EventType       `json:"type"`
	TeamID            *string         `json:"team_id,omitempty"`
	PrimaryPersonID   *string         `json:"primary_person_id,omitempty"`
	SecondaryPersonID *string         `json:"secondary_person_id,omitempty"`
	Detail            *string         `json:"detail,omitempty"`
	HomeScore         *int16          `json:"home_score,omitempty"`
	AwayScore         *int16          `json:"away_score,omitempty"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	OccurredAt        *time.Time      `json:"occurred_at,omitempty"`
}
