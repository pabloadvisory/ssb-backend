package httpx

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
)

type clientIPContextKey struct{}

type ClientIPResolver struct {
	trusted []netip.Prefix
}

func NewClientIPResolver(cidrs []string) (*ClientIPResolver, error) {
	resolver := &ClientIPResolver{trusted: make([]netip.Prefix, 0, len(cidrs))}
	for _, value := range cidrs {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, err
		}
		resolver.trusted = append(resolver.trusted, prefix.Masked())
	}
	return resolver, nil
}

func (resolver *ClientIPResolver) Resolve(request *http.Request) string {
	peer, ok := parseIP(request.RemoteAddr)
	if !ok {
		return "unknown"
	}
	if !resolver.isTrusted(peer) {
		return peer.String()
	}

	forwarded := strings.Split(request.Header.Get("X-Forwarded-For"), ",")
	chain := make([]netip.Addr, 0, len(forwarded))
	for _, value := range forwarded {
		address, valid := parseIP(strings.TrimSpace(value))
		if !valid {
			return peer.String()
		}
		chain = append(chain, address)
	}
	current := peer
	for index := len(chain) - 1; index >= 0; index-- {
		if !resolver.isTrusted(current) {
			return current.String()
		}
		current = chain[index]
	}
	return current.String()
}

func (resolver *ClientIPResolver) isTrusted(address netip.Addr) bool {
	for _, prefix := range resolver.trusted {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func parseIP(value string) (netip.Addr, bool) {
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	address, err := netip.ParseAddr(strings.Trim(value, "[]"))
	if err != nil {
		return netip.Addr{}, false
	}
	return address.Unmap(), true
}

func ResolveClientIP(resolver *ClientIPResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			clientIP := resolver.Resolve(request)
			ctx := context.WithValue(request.Context(), clientIPContextKey{}, clientIP)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

func ClientIPFromContext(ctx context.Context) string {
	value, _ := ctx.Value(clientIPContextKey{}).(string)
	if value == "" {
		return "unknown"
	}
	return value
}

type rateBucket struct {
	tokens   float64
	updated  time.Time
	lastSeen time.Time
}

type RateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*rateBucket
	rate       float64
	burst      float64
	entryTTL   time.Duration
	maxEntries int
	now        func() time.Time
	operations uint64
}

func NewRateLimiter(requests int, window time.Duration, burst, maxEntries int) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*rateBucket), rate: float64(requests) / window.Seconds(),
		burst: float64(burst), entryTTL: 2 * window, maxEntries: maxEntries, now: time.Now,
	}
}

func (limiter *RateLimiter) Allow(key string) (bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now()
	limiter.operations++
	if limiter.operations%256 == 0 {
		limiter.prune(now)
	}
	bucket := limiter.buckets[key]
	if bucket == nil {
		if len(limiter.buckets) >= limiter.maxEntries {
			limiter.prune(now)
			if len(limiter.buckets) >= limiter.maxEntries {
				return false, time.Second
			}
		}
		bucket = &rateBucket{tokens: limiter.burst, updated: now}
		limiter.buckets[key] = bucket
	}

	elapsed := now.Sub(bucket.updated).Seconds()
	if elapsed > 0 {
		bucket.tokens = min(limiter.burst, bucket.tokens+elapsed*limiter.rate)
		bucket.updated = now
	}
	bucket.lastSeen = now
	if bucket.tokens >= 1 {
		bucket.tokens--
		return true, 0
	}
	wait := time.Duration((1-bucket.tokens)/limiter.rate*float64(time.Second)) + time.Millisecond
	return false, max(wait, time.Second)
}

func (limiter *RateLimiter) prune(now time.Time) {
	for key, bucket := range limiter.buckets {
		if now.Sub(bucket.lastSeen) > limiter.entryTTL {
			delete(limiter.buckets, key)
		}
	}
}

type ConnectionLimiter struct {
	mu         sync.Mutex
	counts     map[string]int
	perKey     int
	maxEntries int
}

func NewConnectionLimiter(perKey, maxEntries int) *ConnectionLimiter {
	return &ConnectionLimiter{counts: make(map[string]int), perKey: perKey, maxEntries: maxEntries}
}

func (limiter *ConnectionLimiter) Acquire(key string) (func(), bool) {
	limiter.mu.Lock()
	if limiter.counts[key] >= limiter.perKey || (limiter.counts[key] == 0 && len(limiter.counts) >= limiter.maxEntries) {
		limiter.mu.Unlock()
		return nil, false
	}
	limiter.counts[key]++
	limiter.mu.Unlock()

	var once sync.Once
	release := func() {
		once.Do(func() {
			limiter.mu.Lock()
			limiter.counts[key]--
			if limiter.counts[key] == 0 {
				delete(limiter.counts, key)
			}
			limiter.mu.Unlock()
		})
	}
	return release, true
}
