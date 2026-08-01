package httpx

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIPIgnoresForwardingHeadersFromUntrustedPeer(t *testing.T) {
	t.Parallel()
	resolver, err := NewClientIPResolver([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "203.0.113.9:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.7")
	if got := resolver.Resolve(request); got != "203.0.113.9" {
		t.Fatalf("expected socket peer, got %q", got)
	}
}

func TestClientIPUsesRightmostUntrustedAddress(t *testing.T) {
	t.Parallel()
	resolver, err := NewClientIPResolver([]string{"10.0.0.0/8"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.RemoteAddr = "10.0.0.2:1234"
	request.Header.Set("X-Forwarded-For", "192.0.2.1, 203.0.113.5, 10.0.0.1")
	if got := resolver.Resolve(request); got != "203.0.113.5" {
		t.Fatalf("expected rightmost untrusted address, got %q", got)
	}
}

func TestRateLimiterEnforcesBurstAndRefills(t *testing.T) {
	t.Parallel()
	limiter := NewRateLimiter(60, time.Minute, 2, 10)
	now := time.Unix(1_700_000_000, 0)
	limiter.now = func() time.Time { return now }
	for index := 0; index < 2; index++ {
		if allowed, _ := limiter.Allow("client"); !allowed {
			t.Fatalf("request %d should be allowed", index+1)
		}
	}
	if allowed, _ := limiter.Allow("client"); allowed {
		t.Fatal("request above burst should be denied")
	}
	now = now.Add(time.Second)
	if allowed, _ := limiter.Allow("client"); !allowed {
		t.Fatal("one token should have refilled")
	}
}

func TestConnectionLimiterReleasesCapacityOnce(t *testing.T) {
	t.Parallel()
	limiter := NewConnectionLimiter(1, 10)
	release, ok := limiter.Acquire("client")
	if !ok {
		t.Fatal("first connection should be accepted")
	}
	if _, ok := limiter.Acquire("client"); ok {
		t.Fatal("second connection should be rejected")
	}
	release()
	release()
	if _, ok := limiter.Acquire("client"); !ok {
		t.Fatal("capacity should be restored")
	}
}
