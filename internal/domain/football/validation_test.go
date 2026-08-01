package football

import (
	"errors"
	"testing"
	"time"
)

func TestMatchSnapshotValidation(t *testing.T) {
	t.Parallel()

	valid := MatchSnapshot{
		LeagueID:   "league",
		SeasonID:   "season",
		HomeTeamID: "home",
		AwayTeamID: "away",
		KickoffAt:  time.Now(),
		Status:     MatchScheduled,
		Leg:        1,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid snapshot rejected: %v", err)
	}

	invalid := valid
	invalid.AwayTeamID = invalid.HomeTeamID
	if err := invalid.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}
