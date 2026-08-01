package realtime

import "testing"

func TestHubPublishesOnlyToMatchingSubscribers(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	updates, cancel := hub.Subscribe("match-1")
	defer cancel()

	hub.Publish(Update{MatchID: "match-2", Version: 1})
	select {
	case update := <-updates:
		t.Fatalf("received unrelated update: %+v", update)
	default:
	}

	hub.Publish(Update{MatchID: "match-1", Version: 2})
	update := <-updates
	if update.Version != 2 {
		t.Fatalf("expected version 2, got %d", update.Version)
	}
}

func TestHubSlowSubscriberReceivesLatestUpdate(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	updates, cancel := hub.Subscribe("match-1")
	defer cancel()

	hub.Publish(Update{MatchID: "match-1", Type: "match.updated", Version: 2})
	hub.Publish(Update{MatchID: "match-1", Type: "match.updated", Version: 3})

	update := <-updates
	if update.Version != 3 {
		t.Fatalf("expected latest version 3, got %d", update.Version)
	}
}

func TestHubRejectsVersionRegressionAndListsSubscriptions(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	updates, cancel := hub.Subscribe("match-2")
	defer cancel()
	_, cancelFirst := hub.Subscribe("match-1")
	defer cancelFirst()

	ids := hub.SubscribedMatchIDs()
	if len(ids) != 2 || ids[0] != "match-1" || ids[1] != "match-2" {
		t.Fatalf("unexpected subscribed matches: %v", ids)
	}
	hub.Publish(Update{MatchID: "match-2", Version: 5})
	hub.Publish(Update{MatchID: "match-2", Version: 4})
	if update := <-updates; update.Version != 5 {
		t.Fatalf("expected version 5, got %d", update.Version)
	}
	select {
	case update := <-updates:
		t.Fatalf("received regressed update: %+v", update)
	default:
	}
}
