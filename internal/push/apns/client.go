package apns

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

type Config struct {
	KeyPath string
	KeyID   string
	TeamID  string
}

type Message struct {
	Token       string
	Environment string
	Topic       string
	PushType    string
	Priority    int
	CollapseID  string
	Payload     []byte
}

type Result struct {
	MessageID string
	Reason    string
	Retryable bool
	Invalid   bool
}

type Client struct {
	keyID      string
	teamID     string
	privateKey *ecdsa.PrivateKey
	httpClient *http.Client
	mu         sync.Mutex
	token      string
	tokenAt    time.Time
}

func New(config Config) (*Client, error) {
	keyPEM, err := os.ReadFile(config.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read APNs signing key: %w", err)
	}
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("APNs signing key is not PEM encoded")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse APNs signing key: %w", err)
	}
	privateKey, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || privateKey.Curve != elliptic.P256() {
		return nil, errors.New("APNs signing key must be an ES256 private key")
	}
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	return &Client{
		keyID: config.KeyID, teamID: config.TeamID, privateKey: privateKey,
		httpClient: &http.Client{Transport: transport, Timeout: 15 * time.Second},
	}, nil
}

func (client *Client) Send(ctx context.Context, message Message) (Result, error) {
	providerToken, err := client.providerToken(time.Now())
	if err != nil {
		return Result{}, err
	}
	host := "https://api.push.apple.com"
	if message.Environment == "sandbox" {
		host = "https://api.sandbox.push.apple.com"
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/3/device/"+url.PathEscape(message.Token), bytes.NewReader(message.Payload))
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Authorization", "bearer "+providerToken)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("apns-topic", message.Topic)
	request.Header.Set("apns-push-type", message.PushType)
	request.Header.Set("apns-priority", strconv.Itoa(message.Priority))
	request.Header.Set("apns-expiration", "0")
	if message.CollapseID != "" {
		request.Header.Set("apns-collapse-id", message.CollapseID)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return Result{Retryable: true}, err
	}
	defer response.Body.Close()
	result := Result{MessageID: response.Header.Get("apns-id")}
	if response.StatusCode == http.StatusOK {
		return result, nil
	}
	body, _ := io.ReadAll(io.LimitReader(response.Body, 16<<10))
	var failure struct {
		Reason string `json:"reason"`
	}
	_ = json.Unmarshal(body, &failure)
	result.Reason = failure.Reason
	result.Invalid = response.StatusCode == http.StatusGone || failure.Reason == "BadDeviceToken" || failure.Reason == "DeviceTokenNotForTopic" || failure.Reason == "Unregistered"
	result.Retryable = response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
	return result, fmt.Errorf("APNs returned %s: %s", response.Status, failure.Reason)
}

func (client *Client) providerToken(now time.Time) (string, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.token != "" && now.Sub(client.tokenAt) < 50*time.Minute {
		return client.token, nil
	}
	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": client.keyID})
	claims, _ := json.Marshal(map[string]any{"iss": client.teamID, "iat": now.Unix()})
	unsigned := rawBase64(header) + "." + rawBase64(claims)
	digest := sha256.Sum256([]byte(unsigned))
	r, s, err := ecdsa.Sign(rand.Reader, client.privateKey, digest[:])
	if err != nil {
		return "", fmt.Errorf("sign APNs provider token: %w", err)
	}
	signature := append(paddedBytes(r, 32), paddedBytes(s, 32)...)
	client.token = unsigned + "." + rawBase64(signature)
	client.tokenAt = now
	return client.token, nil
}

func rawBase64(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func paddedBytes(value *big.Int, length int) []byte {
	bytes := value.Bytes()
	result := make([]byte, length)
	copy(result[length-len(bytes):], bytes)
	return result
}
