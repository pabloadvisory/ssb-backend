package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment   string
	LogLevel      slog.Level
	HTTP          HTTP
	Database      Database
	IngestKey     string
	EditorialKey  string
	Push          Push
	Outbox        Outbox
	Maintenance   Maintenance
	Observability Observability
}

type HTTP struct {
	Address                   string
	PublicAssetsDirectory     string
	ReadHeaderTimeout         time.Duration
	ReadTimeout               time.Duration
	WriteTimeout              time.Duration
	IdleTimeout               time.Duration
	ShutdownTimeout           time.Duration
	TrustedProxyCIDRs         []string
	InstallationRatePerMinute int
	InstallationRateBurst     int
	RealtimeConnectionsPerIP  int
	AbuseTrackedIPLimit       int
}

type Database struct {
	URL              string
	MinConnections   int32
	MaxConnections   int32
	HealthTimeout    time.Duration
	ConnectionMaxAge time.Duration
	ConnectionIdle   time.Duration
}

type Push struct {
	APNs         APNs
	FCMProjectID string
	ActivityType string
	PollInterval time.Duration
	LockDuration time.Duration
	SendTimeout  time.Duration
	Concurrency  int
	MaxAttempts  int
}

type APNs struct {
	KeyPath string
	KeyID   string
	TeamID  string
}

type Outbox struct {
	PollInterval   time.Duration
	LockDuration   time.Duration
	ProcessTimeout time.Duration
	Concurrency    int
	MaxAttempts    int
}

type Maintenance struct {
	Interval          time.Duration
	DeliveryRetention time.Duration
	OutboxRetention   time.Duration
	AuditRetention    time.Duration
	BatchSize         int
}

type Observability struct {
	QueueHealthInterval time.Duration
}

func Load(command string) (Config, error) {
	var cfg Config
	var errs []error

	cfg.Environment = env("SSB_ENV", "development")
	cfg.HTTP.Address = env("SSB_HTTP_ADDR", ":8080")
	cfg.HTTP.PublicAssetsDirectory = strings.TrimSpace(os.Getenv("SSB_PUBLIC_ASSETS_DIR"))
	cfg.Database.URL = strings.TrimSpace(os.Getenv("SSB_DATABASE_URL"))
	cfg.IngestKey = os.Getenv("SSB_INGEST_API_KEY")
	cfg.EditorialKey = os.Getenv("SSB_EDITORIAL_API_KEY")
	cfg.Push.APNs.KeyPath = strings.TrimSpace(os.Getenv("SSB_APNS_KEY_PATH"))
	cfg.Push.APNs.KeyID = strings.TrimSpace(os.Getenv("SSB_APNS_KEY_ID"))
	cfg.Push.APNs.TeamID = strings.TrimSpace(os.Getenv("SSB_APNS_TEAM_ID"))
	cfg.Push.FCMProjectID = strings.TrimSpace(os.Getenv("SSB_FCM_PROJECT_ID"))
	cfg.Push.ActivityType = env("SSB_ACTIVITY_ATTRIBUTES_TYPE", "FootballMatchAttributes")

	cfg.LogLevel = parseLogLevel(env("SSB_LOG_LEVEL", "info"), &errs)
	cfg.HTTP.ReadHeaderTimeout = duration("SSB_HTTP_READ_HEADER_TIMEOUT", 5*time.Second, &errs)
	cfg.HTTP.ReadTimeout = duration("SSB_HTTP_READ_TIMEOUT", 15*time.Second, &errs)
	cfg.HTTP.WriteTimeout = duration("SSB_HTTP_WRITE_TIMEOUT", 30*time.Second, &errs)
	cfg.HTTP.IdleTimeout = duration("SSB_HTTP_IDLE_TIMEOUT", 2*time.Minute, &errs)
	cfg.HTTP.ShutdownTimeout = duration("SSB_SHUTDOWN_TIMEOUT", 20*time.Second, &errs)
	cfg.HTTP.TrustedProxyCIDRs = csv("SSB_TRUSTED_PROXY_CIDRS")
	cfg.HTTP.InstallationRatePerMinute = intValue("SSB_INSTALLATION_RATE_PER_MINUTE", 20, &errs)
	cfg.HTTP.InstallationRateBurst = intValue("SSB_INSTALLATION_RATE_BURST", 5, &errs)
	cfg.HTTP.RealtimeConnectionsPerIP = intValue("SSB_REALTIME_CONNECTIONS_PER_IP", 20, &errs)
	cfg.HTTP.AbuseTrackedIPLimit = intValue("SSB_ABUSE_TRACKED_IP_LIMIT", 100_000, &errs)

	cfg.Database.MinConnections = int32Value("SSB_DATABASE_MIN_CONNS", 2, &errs)
	cfg.Database.MaxConnections = int32Value("SSB_DATABASE_MAX_CONNS", 20, &errs)
	cfg.Database.HealthTimeout = duration("SSB_DATABASE_HEALTH_TIMEOUT", 2*time.Second, &errs)
	cfg.Database.ConnectionMaxAge = duration("SSB_DATABASE_CONN_MAX_LIFETIME", 30*time.Minute, &errs)
	cfg.Database.ConnectionIdle = duration("SSB_DATABASE_CONN_MAX_IDLE_TIME", 5*time.Minute, &errs)
	cfg.Push.PollInterval = duration("SSB_PUSH_POLL_INTERVAL", time.Second, &errs)
	cfg.Push.LockDuration = duration("SSB_PUSH_LOCK_DURATION", 30*time.Second, &errs)
	cfg.Push.SendTimeout = duration("SSB_PUSH_SEND_TIMEOUT", 20*time.Second, &errs)
	cfg.Push.Concurrency = intValue("SSB_PUSH_CONCURRENCY", 10, &errs)
	cfg.Push.MaxAttempts = intValue("SSB_PUSH_MAX_ATTEMPTS", 8, &errs)
	cfg.Outbox.PollInterval = duration("SSB_OUTBOX_POLL_INTERVAL", 250*time.Millisecond, &errs)
	cfg.Outbox.LockDuration = duration("SSB_OUTBOX_LOCK_DURATION", 150*time.Second, &errs)
	cfg.Outbox.ProcessTimeout = duration("SSB_OUTBOX_PROCESS_TIMEOUT", 2*time.Minute, &errs)
	cfg.Outbox.Concurrency = intValue("SSB_OUTBOX_CONCURRENCY", 4, &errs)
	cfg.Outbox.MaxAttempts = intValue("SSB_OUTBOX_MAX_ATTEMPTS", 10, &errs)
	cfg.Maintenance.Interval = duration("SSB_MAINTENANCE_INTERVAL", time.Hour, &errs)
	cfg.Maintenance.DeliveryRetention = duration("SSB_DELIVERY_RETENTION", 30*24*time.Hour, &errs)
	cfg.Maintenance.OutboxRetention = duration("SSB_OUTBOX_RETENTION", 7*24*time.Hour, &errs)
	cfg.Maintenance.AuditRetention = duration("SSB_ENDPOINT_AUDIT_RETENTION", 365*24*time.Hour, &errs)
	cfg.Maintenance.BatchSize = intValue("SSB_MAINTENANCE_BATCH_SIZE", 1000, &errs)
	cfg.Observability.QueueHealthInterval = duration("SSB_QUEUE_HEALTH_INTERVAL", 30*time.Second, &errs)

	if command == "serve" || command == "migrate" || command == "push-worker" || command == "seed" {
		if cfg.Database.URL == "" {
			errs = append(errs, errors.New("SSB_DATABASE_URL is required"))
		}
	}
	if cfg.Database.MinConnections < 0 {
		errs = append(errs, errors.New("SSB_DATABASE_MIN_CONNS must not be negative"))
	}
	if cfg.Database.MaxConnections < 1 {
		errs = append(errs, errors.New("SSB_DATABASE_MAX_CONNS must be positive"))
	}
	if cfg.Database.MinConnections > cfg.Database.MaxConnections {
		errs = append(errs, errors.New("SSB_DATABASE_MIN_CONNS must not exceed SSB_DATABASE_MAX_CONNS"))
	}
	for _, value := range cfg.HTTP.TrustedProxyCIDRs {
		if _, err := netip.ParsePrefix(value); err != nil {
			errs = append(errs, fmt.Errorf("SSB_TRUSTED_PROXY_CIDRS contains invalid prefix %q", value))
		}
	}
	if cfg.HTTP.InstallationRatePerMinute < 1 {
		errs = append(errs, errors.New("SSB_INSTALLATION_RATE_PER_MINUTE must be positive"))
	}
	if cfg.HTTP.InstallationRateBurst < 1 {
		errs = append(errs, errors.New("SSB_INSTALLATION_RATE_BURST must be positive"))
	}
	if cfg.HTTP.RealtimeConnectionsPerIP < 1 {
		errs = append(errs, errors.New("SSB_REALTIME_CONNECTIONS_PER_IP must be positive"))
	}
	if cfg.HTTP.AbuseTrackedIPLimit < 1 {
		errs = append(errs, errors.New("SSB_ABUSE_TRACKED_IP_LIMIT must be positive"))
	}
	if command == "serve" && cfg.Environment == "production" && len(cfg.IngestKey) < 32 {
		errs = append(errs, errors.New("SSB_INGEST_API_KEY must contain at least 32 characters in production"))
	}
	if command == "serve" && cfg.Environment == "production" && cfg.EditorialKey != "" && len(cfg.EditorialKey) < 32 {
		errs = append(errs, errors.New("SSB_EDITORIAL_API_KEY must contain at least 32 characters when configured in production"))
	}
	apnsValues := []string{cfg.Push.APNs.KeyPath, cfg.Push.APNs.KeyID, cfg.Push.APNs.TeamID}
	apnsConfigured := 0
	for _, value := range apnsValues {
		if value != "" {
			apnsConfigured++
		}
	}
	if apnsConfigured != 0 && apnsConfigured != len(apnsValues) {
		errs = append(errs, errors.New("SSB_APNS_KEY_PATH, SSB_APNS_KEY_ID, and SSB_APNS_TEAM_ID must be configured together"))
	}
	if command == "push-worker" && apnsConfigured == 0 && cfg.Push.FCMProjectID == "" {
		errs = append(errs, errors.New("push-worker requires APNs credentials, SSB_FCM_PROJECT_ID, or both"))
	}
	if cfg.Push.Concurrency < 1 || cfg.Push.Concurrency > 100 {
		errs = append(errs, errors.New("SSB_PUSH_CONCURRENCY must be between 1 and 100"))
	}
	if cfg.Push.LockDuration < cfg.Push.SendTimeout+5*time.Second {
		errs = append(errs, errors.New("SSB_PUSH_LOCK_DURATION must be at least SSB_PUSH_SEND_TIMEOUT plus 5s"))
	}
	if cfg.Push.MaxAttempts < 1 || cfg.Push.MaxAttempts > 100 {
		errs = append(errs, errors.New("SSB_PUSH_MAX_ATTEMPTS must be between 1 and 100"))
	}
	if cfg.Outbox.Concurrency < 1 || cfg.Outbox.Concurrency > 32 {
		errs = append(errs, errors.New("SSB_OUTBOX_CONCURRENCY must be between 1 and 32"))
	}
	if cfg.Outbox.LockDuration < cfg.Outbox.ProcessTimeout+5*time.Second {
		errs = append(errs, errors.New("SSB_OUTBOX_LOCK_DURATION must be at least SSB_OUTBOX_PROCESS_TIMEOUT plus 5s"))
	}
	if cfg.Outbox.MaxAttempts < 1 || cfg.Outbox.MaxAttempts > 100 {
		errs = append(errs, errors.New("SSB_OUTBOX_MAX_ATTEMPTS must be between 1 and 100"))
	}
	if cfg.Maintenance.BatchSize < 1 || cfg.Maintenance.BatchSize > 10_000 {
		errs = append(errs, errors.New("SSB_MAINTENANCE_BATCH_SIZE must be between 1 and 10000"))
	}

	return cfg, errors.Join(errs...)
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func csv(key string) []string {
	values := strings.Split(os.Getenv(key), ",")
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func duration(key string, fallback time.Duration, errs *[]error) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		*errs = append(*errs, fmt.Errorf("%s must be a positive duration", key))
		return fallback
	}
	return parsed
}

func int32Value(key string, fallback int32, errs *[]error) int32 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s must be a valid integer", key))
		return fallback
	}
	return int32(parsed)
}

func intValue(key string, fallback int, errs *[]error) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		*errs = append(*errs, fmt.Errorf("%s must be a valid integer", key))
		return fallback
	}
	return parsed
}

func parseLogLevel(value string, errs *[]error) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		*errs = append(*errs, fmt.Errorf("SSB_LOG_LEVEL must be debug, info, warn, or error"))
		return slog.LevelInfo
	}
	return level
}
