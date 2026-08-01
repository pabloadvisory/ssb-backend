package football

import (
	"fmt"
	"strings"
)

func (snapshot MatchSnapshot) Validate() error {
	var problems []string
	if snapshot.LeagueID == "" {
		problems = append(problems, "league_id is required")
	}
	if snapshot.SeasonID == "" {
		problems = append(problems, "season_id is required")
	}
	if snapshot.HomeTeamID == "" || snapshot.AwayTeamID == "" {
		problems = append(problems, "home_team_id and away_team_id are required")
	}
	if snapshot.HomeTeamID != "" && snapshot.HomeTeamID == snapshot.AwayTeamID {
		problems = append(problems, "home and away teams must differ")
	}
	if snapshot.KickoffAt.IsZero() {
		problems = append(problems, "kickoff_at is required")
	}
	if !snapshot.Status.Valid() {
		problems = append(problems, "status is invalid")
	}
	if snapshot.Leg < 1 {
		problems = append(problems, "leg must be positive")
	}
	if snapshot.Period != nil && !snapshot.Period.Valid() {
		problems = append(problems, "period is invalid")
	}
	if snapshot.RoundSort != nil && *snapshot.RoundSort < 1 {
		problems = append(problems, "round_sort must be positive")
	}
	if snapshot.Attendance != nil && *snapshot.Attendance < 0 {
		problems = append(problems, "attendance cannot be negative")
	}
	for field, score := range map[string]*int16{
		"home_score": snapshot.HomeScore, "away_score": snapshot.AwayScore,
		"home_half_time_score": snapshot.HomeHTScore, "away_half_time_score": snapshot.AwayHTScore,
		"home_extra_time_score": snapshot.HomeExtraTimeScore, "away_extra_time_score": snapshot.AwayExtraTimeScore,
		"home_penalty_score": snapshot.HomePenaltyScore, "away_penalty_score": snapshot.AwayPenaltyScore,
	} {
		if score != nil && *score < 0 {
			problems = append(problems, field+" cannot be negative")
		}
	}
	sequences := make(map[int]struct{}, len(snapshot.Events))
	for index, event := range snapshot.Events {
		if event.Sequence < 1 {
			problems = append(problems, fmt.Sprintf("events[%d].sequence must be positive", index))
		}
		if !event.Type.Valid() {
			problems = append(problems, fmt.Sprintf("events[%d].type is invalid", index))
		}
		if event.Period != nil && !event.Period.Valid() {
			problems = append(problems, fmt.Sprintf("events[%d].period is invalid", index))
		}
		if event.HomeScore != nil && *event.HomeScore < 0 {
			problems = append(problems, fmt.Sprintf("events[%d].home_score cannot be negative", index))
		}
		if event.AwayScore != nil && *event.AwayScore < 0 {
			problems = append(problems, fmt.Sprintf("events[%d].away_score cannot be negative", index))
		}
		if _, duplicate := sequences[event.Sequence]; duplicate {
			problems = append(problems, fmt.Sprintf("events[%d].sequence is duplicated", index))
		}
		sequences[event.Sequence] = struct{}{}
	}
	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrInvalid, strings.Join(problems, "; "))
	}
	return nil
}

func (period MatchPeriod) Valid() bool {
	switch period {
	case PeriodFirstHalf, PeriodHalfTime, PeriodSecondHalf,
		PeriodExtraTimeFirstHalf, PeriodExtraTimeHalfTime, PeriodExtraTimeSecondHalf,
		PeriodPenalties, PeriodFullTime:
		return true
	default:
		return false
	}
}

func (eventType EventType) Valid() bool {
	switch eventType {
	case EventKickoff, EventHalfTime, EventSecondHalf, EventExtraTime, EventExtraTimeHalf,
		EventPenaltiesStarted, EventFullTime, EventGoal, EventOwnGoal, EventPenaltyGoal,
		EventPenaltyMissed, EventYellowCard, EventSecondYellow, EventRedCard,
		EventSubstitution, EventVARDecision, EventMatchSuspended, EventMatchResumed,
		EventMatchCancelled:
		return true
	default:
		return false
	}
}

func (status MatchStatus) Valid() bool {
	switch status {
	case MatchScheduled, MatchPostponed, MatchCancelled, MatchLive, MatchSuspended, MatchFinished, MatchAbandoned, MatchAwarded:
		return true
	default:
		return false
	}
}
