package postgres

import (
	"testing"

	"github.com/pabloadvisory/ssb-backend/internal/domain/football"
)

func TestCanonicalizePlayerSpatialRotatesRightToLeftCoordinates(t *testing.T) {
	t.Parallel()

	command := football.UpsertPlayerSpatial{
		Orientation: "attacking_right_to_left",
		Touches:     []football.TouchPointInput{{Sequence: 1, X: 80, Y: 25, Intensity: 1}},
		Shots:       []football.ShotInput{{Sequence: 1, X: 92, Y: 40, ExpectedGoals: 0.5}},
	}
	canonical, err := canonicalizePlayerSpatial(command)
	if err != nil {
		t.Fatalf("canonicalize spatial input: %v", err)
	}
	if canonical.Orientation != canonicalPlayerSpatialOrientation {
		t.Fatalf("unexpected orientation: %q", canonical.Orientation)
	}
	if canonical.Touches[0].X != 20 || canonical.Touches[0].Y != 75 {
		t.Fatalf("touch was not rotated: %+v", canonical.Touches[0])
	}
	if canonical.Shots[0].X != 8 || canonical.Shots[0].Y != 60 {
		t.Fatalf("shot was not rotated: %+v", canonical.Shots[0])
	}
}

func TestCanonicalizePlayerSpatialPreservesCanonicalCoordinates(t *testing.T) {
	t.Parallel()

	command := football.UpsertPlayerSpatial{
		Orientation: canonicalPlayerSpatialOrientation,
		Touches:     []football.TouchPointInput{{Sequence: 1, X: 20, Y: 75, Intensity: 1}},
	}
	canonical, err := canonicalizePlayerSpatial(command)
	if err != nil {
		t.Fatalf("canonicalize spatial input: %v", err)
	}
	if canonical.Touches[0].X != 20 || canonical.Touches[0].Y != 75 {
		t.Fatalf("canonical input changed: %+v", canonical.Touches[0])
	}
}

func TestBoundedPlayerShotLimit(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		input int
		want  int
	}{{0, 100}, {-1, 100}, {25, 25}, {500, 200}} {
		if got := boundedPlayerShotLimit(test.input); got != test.want {
			t.Fatalf("boundedPlayerShotLimit(%d) = %d, want %d", test.input, got, test.want)
		}
	}
}
