package eventing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/realtime"
)

const MatchChangedV1 = "match.changed.v1"

var ErrLeaseLost = errors.New("outbox event lease is no longer owned")

type Event struct {
	ID        string
	Type      string
	Payload   json.RawMessage
	LockToken string
	Attempts  int
}

type MatchChanged struct {
	Realtime     realtime.Update          `json:"realtime"`
	Notification notification.MatchChange `json:"notification"`
}

type Store interface {
	ClaimOutboxEvents(context.Context, int, time.Duration) ([]Event, error)
	PublishMatchChanged(context.Context, Event, MatchChanged, []notification.DeliveryPlan) error
	RetryOutboxEvent(context.Context, string, string, time.Time, string, bool) error
}
