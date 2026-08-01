package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/notification"
	"github.com/pabloadvisory/ssb-backend/internal/push/apns"
	"github.com/pabloadvisory/ssb-backend/internal/push/fcm"
)

type Result struct {
	MessageID string
	Reason    string
	Retryable bool
	Invalid   bool
}

type Dispatcher struct {
	apns         *apns.Client
	fcm          *fcm.Client
	activityType string
}

func NewDispatcher(apnsClient *apns.Client, fcmClient *fcm.Client, activityType string) *Dispatcher {
	return &Dispatcher{apns: apnsClient, fcm: fcmClient, activityType: activityType}
}

func (dispatcher *Dispatcher) Send(ctx context.Context, delivery notification.Delivery) (Result, error) {
	var update notification.MatchUpdate
	if err := json.Unmarshal(delivery.Payload, &update); err != nil {
		return Result{Reason: "invalid queued payload"}, err
	}
	switch delivery.Transport {
	case notification.TransportAPNs:
		return dispatcher.sendAPNs(ctx, delivery, update)
	case notification.TransportFCM:
		return dispatcher.sendFCM(ctx, delivery, update)
	default:
		return Result{Reason: "unsupported push transport"}, errors.New("unsupported push transport")
	}
}

func (dispatcher *Dispatcher) sendAPNs(ctx context.Context, delivery notification.Delivery, update notification.MatchUpdate) (Result, error) {
	if dispatcher.apns == nil {
		return Result{Reason: "APNs provider is not configured", Retryable: true}, errors.New("APNs provider is not configured")
	}
	pushType := "alert"
	topic := delivery.AppID
	priority := 10
	payload, err := standardPayload(update)
	if err != nil {
		return Result{}, err
	}
	if delivery.Kind == notification.DeliveryLiveActivityStart || delivery.Kind == notification.DeliveryLiveActivityUpdate || delivery.Kind == notification.DeliveryLiveActivityEnd {
		pushType = "liveactivity"
		topic += ".push-type.liveactivity"
		priority = 5
		if delivery.Priority == notification.PriorityHigh {
			priority = 10
		}
		payload, err = liveActivityPayload(delivery.Kind, dispatcher.activityType, update)
		if err != nil {
			return Result{}, err
		}
	}
	collapseID := ""
	if delivery.CollapseKey != nil {
		collapseID = *delivery.CollapseKey
	}
	result, err := dispatcher.apns.Send(ctx, apns.Message{
		Token: delivery.Token, Environment: delivery.Environment, Topic: topic,
		PushType: pushType, Priority: priority, CollapseID: collapseID, Payload: payload,
	})
	return Result(result), err
}

func (dispatcher *Dispatcher) sendFCM(ctx context.Context, delivery notification.Delivery, update notification.MatchUpdate) (Result, error) {
	if dispatcher.fcm == nil {
		return Result{Reason: "FCM provider is not configured", Retryable: true}, errors.New("FCM provider is not configured")
	}
	if delivery.Kind != notification.DeliveryMatchUpdate && delivery.Kind != notification.DeliveryMatchFinished {
		return Result{Reason: "ActivityKit delivery cannot target FCM"}, errors.New("ActivityKit delivery cannot target FCM")
	}
	priority := "normal"
	if delivery.Priority == notification.PriorityHigh {
		priority = "high"
	}
	collapseKey := ""
	if delivery.CollapseKey != nil {
		collapseKey = *delivery.CollapseKey
	}
	result, err := dispatcher.fcm.Send(ctx, fcm.Message{
		Token: delivery.Token, Title: update.HomeTeamName + " vs " + update.AwayTeamName,
		Body: scoreBody(update), Priority: priority, CollapseKey: collapseKey, TTL: 2 * time.Minute,
		Data: map[string]string{
			"type": "match_update", "match_id": update.MatchID, "version": strconv.FormatInt(update.Version, 10),
			"status": string(update.Status),
		},
	})
	return Result(result), err
}

func standardPayload(update notification.MatchUpdate) ([]byte, error) {
	return json.Marshal(map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{"title": update.HomeTeamName + " vs " + update.AwayTeamName, "body": scoreBody(update)},
			"sound": "default", "thread-id": "match:" + update.MatchID,
		},
		"type": "match_update", "match_id": update.MatchID, "version": update.Version,
	})
}

func liveActivityPayload(kind notification.DeliveryKind, attributesType string, update notification.MatchUpdate) ([]byte, error) {
	event := "update"
	if kind == notification.DeliveryLiveActivityStart {
		event = "start"
	} else if kind == notification.DeliveryLiveActivityEnd {
		event = "end"
	}
	contentState := map[string]any{
		"homeScore": update.HomeScore, "awayScore": update.AwayScore, "status": update.Status,
		"elapsedMinute": update.ElapsedMinute, "period": update.Period,
	}
	aps := map[string]any{
		"timestamp": time.Now().Unix(), "event": event, "content-state": contentState,
		"stale-date": time.Now().Add(90 * time.Second).Unix(),
	}
	if event == "start" {
		aps["attributes-type"] = attributesType
		aps["attributes"] = map[string]any{
			"matchID": update.MatchID, "homeTeamName": update.HomeTeamName, "awayTeamName": update.AwayTeamName,
		}
		aps["input-push-token"] = 1
		aps["alert"] = map[string]string{"title": update.HomeTeamName + " vs " + update.AwayTeamName, "body": "The match is live"}
	}
	if event == "end" {
		aps["dismissal-date"] = time.Now().Add(4 * time.Hour).Unix()
	}
	return json.Marshal(map[string]any{"aps": aps})
}

func scoreBody(update notification.MatchUpdate) string {
	if update.HomeScore == nil || update.AwayScore == nil {
		return fmt.Sprintf("Match status: %s", update.Status)
	}
	return fmt.Sprintf("%d – %d · %s", *update.HomeScore, *update.AwayScore, update.Status)
}
