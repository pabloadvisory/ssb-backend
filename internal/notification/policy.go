package notification

import "github.com/pabloadvisory/ssb-backend/internal/domain/football"

type MatchState struct {
	Status    football.MatchStatus `json:"status"`
	HomeScore *int16               `json:"home_score,omitempty"`
	AwayScore *int16               `json:"away_score,omitempty"`
}

type MatchChange struct {
	Existed  bool        `json:"existed"`
	Previous MatchState  `json:"previous"`
	Current  MatchUpdate `json:"current"`
}

type Audience string

const (
	AudienceLiveActivities Audience = "live_activities"
	AudienceSubscribers    Audience = "subscribers"
	AudiencePushToStart    Audience = "push_to_start"
)

type DeliveryPlan struct {
	Audience          Audience
	Kind              DeliveryKind
	Priority          Priority
	IdempotencyPrefix string
}

// PlanMatchDeliveries is deliberately pure so notification policy can evolve independently of SQL transactions.
func PlanMatchDeliveries(change MatchChange) []DeliveryPlan {
	significant := change.Existed && (change.Previous.Status != change.Current.Status ||
		scoreChanged(change.Previous.HomeScore, change.Current.HomeScore) ||
		scoreChanged(change.Previous.AwayScore, change.Current.AwayScore))

	liveKind := DeliveryLiveActivityUpdate
	currentStatus := change.Current.Status
	if currentStatus == football.MatchFinished || currentStatus == football.MatchCancelled || currentStatus == football.MatchAbandoned {
		liveKind = DeliveryLiveActivityEnd
	}
	livePriority := PriorityNormal
	if significant {
		livePriority = PriorityHigh
	}
	plans := []DeliveryPlan{{
		Audience: AudienceLiveActivities, Kind: liveKind,
		Priority: livePriority, IdempotencyPrefix: "live",
	}}

	if significant {
		kind := DeliveryMatchUpdate
		if currentStatus == football.MatchFinished {
			kind = DeliveryMatchFinished
		}
		plans = append(plans, DeliveryPlan{
			Audience: AudienceSubscribers, Kind: kind,
			Priority: PriorityHigh, IdempotencyPrefix: "standard",
		})
	}
	if change.Existed && change.Previous.Status != football.MatchLive && currentStatus == football.MatchLive {
		plans = append(plans, DeliveryPlan{
			Audience: AudiencePushToStart, Kind: DeliveryLiveActivityStart,
			Priority: PriorityHigh, IdempotencyPrefix: "live-start",
		})
	}
	return plans
}

func scoreChanged(previous, current *int16) bool {
	if previous == nil || current == nil {
		return previous != nil || current != nil
	}
	return *previous != *current
}
