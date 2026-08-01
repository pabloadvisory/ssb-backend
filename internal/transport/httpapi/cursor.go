package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type leagueCursor struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type matchCursor struct {
	KickoffAt time.Time `json:"kickoff_at"`
	ID        string    `json:"id"`
}

func encodeCursor(value any) string {
	encoded, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func decodeCursor(value string, destination any) error {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf("cursor is not valid base64")
	}
	if err := json.Unmarshal(decoded, destination); err != nil {
		return fmt.Errorf("cursor has an invalid shape")
	}
	return nil
}

func parseLimit(value string, fallback, maximum int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 || parsed > maximum {
		return 0, fmt.Errorf("limit must be between 1 and %d", maximum)
	}
	return parsed, nil
}

func parseTime(value string, endOfDay bool) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, fmt.Errorf("date must be YYYY-MM-DD or RFC3339")
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
