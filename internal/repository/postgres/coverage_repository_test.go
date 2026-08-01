package postgres

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func TestCoverageMetadataRecordsAuthoritativeSource(t *testing.T) {
	t.Parallel()

	metadata, err := coverageMetadata(json.RawMessage(`{"note":"provider value","ingestion_source":"stale"}`), "feed-a")
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(metadata, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["note"] != "provider value" || decoded["ingestion_source"] != "feed-a" {
		t.Fatalf("unexpected metadata: %v", decoded)
	}

	if _, err := coverageMetadata(json.RawMessage(`[]`), "feed-a"); !errors.Is(err, football.ErrInvalid) {
		t.Fatalf("non-object metadata must be invalid, got %v", err)
	}
}

func TestCoverageParticipantValidationChecksEveryTeamDataset(t *testing.T) {
	t.Parallel()

	nonParticipant := "team-c"
	lineups := []football.LineupInput{{TeamID: nonParticipant}}
	update := football.MatchCoverageUpdate{Lineups: &lineups}
	err := validateCoverageParticipants(update, func(teamID string) bool {
		return teamID == "team-a" || teamID == "team-b"
	})
	if !errors.Is(err, football.ErrInvalid) {
		t.Fatalf("expected invalid participant error, got %v", err)
	}
}
