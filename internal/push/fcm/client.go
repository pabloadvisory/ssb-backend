package fcm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const messagingScope = "https://www.googleapis.com/auth/firebase.messaging"

type Message struct {
	Token       string
	Title       string
	Body        string
	Data        map[string]string
	Priority    string
	CollapseKey string
	TTL         time.Duration
}

type Result struct {
	MessageID string
	Reason    string
	Retryable bool
	Invalid   bool
}

type Client struct {
	projectID  string
	httpClient *http.Client
}

func New(ctx context.Context, projectID string) (*Client, error) {
	credentials, err := google.FindDefaultCredentials(ctx, messagingScope)
	if err != nil {
		return nil, fmt.Errorf("load Google application credentials: %w", err)
	}
	httpClient := oauth2.NewClient(ctx, credentials.TokenSource)
	httpClient.Timeout = 15 * time.Second
	return &Client{projectID: projectID, httpClient: httpClient}, nil
}

func (client *Client) Send(ctx context.Context, message Message) (Result, error) {
	body := map[string]any{
		"message": map[string]any{
			"token": message.Token,
			"notification": map[string]string{
				"title": message.Title,
				"body":  message.Body,
			},
			"data": message.Data,
			"android": map[string]any{
				"priority":     message.Priority,
				"collapse_key": message.CollapseKey,
				"ttl":          fmt.Sprintf("%ds", int64(message.TTL.Seconds())),
			},
		},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}
	endpoint := "https://fcm.googleapis.com/v1/projects/" + url.PathEscape(client.projectID) + "/messages:send"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return Result{Retryable: true}, err
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, 32<<10))
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		var success struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(responseBody, &success)
		return Result{MessageID: success.Name}, nil
	}
	failure := decodeFailure(responseBody)
	reason := failure.Error.Status
	if failure.Error.Message != "" {
		reason += ": " + failure.Error.Message
	}
	result := Result{Reason: reason}
	result.Invalid = failure.invalidEndpoint()
	result.Retryable = response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500 || failure.Error.Status == "UNAVAILABLE"
	return result, fmt.Errorf("FCM returned %s: %s", response.Status, reason)
}

type failureResponse struct {
	Error struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Details []struct {
			Type      string `json:"@type"`
			ErrorCode string `json:"errorCode"`
		} `json:"details"`
	} `json:"error"`
}

func decodeFailure(body []byte) failureResponse {
	var failure failureResponse
	_ = json.Unmarshal(body, &failure)
	return failure
}

func (failure failureResponse) invalidEndpoint() bool {
	for _, detail := range failure.Error.Details {
		if detail.ErrorCode == "UNREGISTERED" {
			return true
		}
	}
	return false
}
