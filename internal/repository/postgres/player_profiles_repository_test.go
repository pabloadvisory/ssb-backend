package postgres

import (
	"testing"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func TestTeamType(t *testing.T) {
	if got := teamType(false); got != "club" {
		t.Fatalf("expected club, got %q", got)
	}
	if got := teamType(true); got != "national" {
		t.Fatalf("expected national, got %q", got)
	}
}

func TestPlayerMatchResult(t *testing.T) {
	home, away := int16(2), int16(1)
	tests := []struct {
		name    string
		match   football.Match
		teamID  string
		expect  string
		isKnown bool
	}{
		{name: "unfinished", match: football.Match{Status: football.MatchLive, HomeTeamID: "home", AwayTeamID: "away", HomeScore: &home, AwayScore: &away}, teamID: "home"},
		{name: "home win", match: football.Match{Status: football.MatchFinished, HomeTeamID: "home", AwayTeamID: "away", HomeScore: &home, AwayScore: &away}, teamID: "home", expect: "win", isKnown: true},
		{name: "away loss", match: football.Match{Status: football.MatchFinished, HomeTeamID: "home", AwayTeamID: "away", HomeScore: &home, AwayScore: &away}, teamID: "away", expect: "loss", isKnown: true},
		{name: "awarded winner", match: football.Match{Status: football.MatchAwarded, HomeTeamID: "home", AwayTeamID: "away", WinnerTeamID: stringPointer("away")}, teamID: "away", expect: "win", isKnown: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := playerMatchResult(test.match, test.teamID)
			if !test.isKnown {
				if got != nil {
					t.Fatalf("expected unknown result, got %q", *got)
				}
				return
			}
			if got == nil || *got != test.expect {
				t.Fatalf("expected %q, got %v", test.expect, got)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }

func TestComparisonMetricsAreStable(t *testing.T) {
	if len(comparisonMetrics) != 23 {
		t.Fatalf("unexpected comparison metric count: %d", len(comparisonMetrics))
	}
	if comparisonMetrics[0] != "appearances" || comparisonMetrics[len(comparisonMetrics)-1] != "average_rating" {
		t.Fatalf("unexpected metric ordering: %v", comparisonMetrics)
	}
}
