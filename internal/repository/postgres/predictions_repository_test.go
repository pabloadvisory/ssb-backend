package postgres

import "testing"

func TestPredictionPercentagesAreDeterministicAndSumToOneHundred(t *testing.T) {
	t.Parallel()

	percentages := predictionPercentages([3]int64{1, 1, 1})
	if percentages != [3]float64{33.34, 33.33, 33.33} {
		t.Fatalf("unexpected tied percentages: %v", percentages)
	}
	if percentages[0]+percentages[1]+percentages[2] != 100 {
		t.Fatalf("percentages must total 100: %v", percentages)
	}

	empty := predictionPercentages([3]int64{})
	if empty != [3]float64{} {
		t.Fatalf("empty poll must contain zero percentages: %v", empty)
	}
}
