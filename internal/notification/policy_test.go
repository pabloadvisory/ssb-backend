package notification

import (
	"testing"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func TestPlanMatchDeliveries(t *testing.T) {
	t.Parallel()
	one, two := int16(1), int16(2)
	tests := []struct {
		name   string
		change MatchChange
		kinds  []DeliveryKind
	}{
		{
			name:   "new match only updates existing activities",
			change: MatchChange{Current: MatchUpdate{Status: football.MatchScheduled}},
			kinds:  []DeliveryKind{DeliveryLiveActivityUpdate},
		},
		{
			name: "kickoff updates activities subscribers and push to start",
			change: MatchChange{
				Existed: true, Previous: MatchState{Status: football.MatchScheduled},
				Current: MatchUpdate{Status: football.MatchLive},
			},
			kinds: []DeliveryKind{DeliveryLiveActivityUpdate, DeliveryMatchUpdate, DeliveryLiveActivityStart},
		},
		{
			name: "score change is significant",
			change: MatchChange{
				Existed: true, Previous: MatchState{Status: football.MatchLive, HomeScore: &one},
				Current: MatchUpdate{Status: football.MatchLive, HomeScore: &two},
			},
			kinds: []DeliveryKind{DeliveryLiveActivityUpdate, DeliveryMatchUpdate},
		},
		{
			name: "finish ends activity and sends final notification",
			change: MatchChange{
				Existed: true, Previous: MatchState{Status: football.MatchLive},
				Current: MatchUpdate{Status: football.MatchFinished},
			},
			kinds: []DeliveryKind{DeliveryLiveActivityEnd, DeliveryMatchFinished},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			plans := PlanMatchDeliveries(test.change)
			if len(plans) != len(test.kinds) {
				t.Fatalf("expected %d plans, got %+v", len(test.kinds), plans)
			}
			for index, kind := range test.kinds {
				if plans[index].Kind != kind {
					t.Fatalf("plan %d: expected %q, got %q", index, kind, plans[index].Kind)
				}
			}
		})
	}
}
