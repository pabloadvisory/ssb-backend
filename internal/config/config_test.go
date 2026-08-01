package config

import "testing"

func TestLoadRequiresDatabaseURLForServe(t *testing.T) {
	t.Setenv("SSB_DATABASE_URL", "")

	_, err := Load("serve")
	if err == nil {
		t.Fatal("expected missing database URL to fail")
	}
}

func TestLoadAllowsHealthcheckWithoutDatabaseURL(t *testing.T) {
	t.Setenv("SSB_DATABASE_URL", "")

	if _, err := Load("healthcheck"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProductionRequiresStrongIngestKey(t *testing.T) {
	t.Setenv("SSB_ENV", "production")
	t.Setenv("SSB_DATABASE_URL", "postgres://example")
	t.Setenv("SSB_INGEST_API_KEY", "short")

	_, err := Load("serve")
	if err == nil {
		t.Fatal("expected a short production ingest key to fail")
	}
}

func TestProductionRejectsWeakConfiguredEditorialKey(t *testing.T) {
	t.Setenv("SSB_ENV", "production")
	t.Setenv("SSB_DATABASE_URL", "postgres://example")
	t.Setenv("SSB_INGEST_API_KEY", "a-production-ingest-key-that-is-long-enough")
	t.Setenv("SSB_EDITORIAL_API_KEY", "short")

	_, err := Load("serve")
	if err == nil {
		t.Fatal("expected a short configured production editorial key to fail")
	}
}

func TestLoadRejectsInvalidTrustedProxyCIDR(t *testing.T) {
	t.Setenv("SSB_DATABASE_URL", "")
	t.Setenv("SSB_TRUSTED_PROXY_CIDRS", "not-a-network")

	if _, err := Load("healthcheck"); err == nil {
		t.Fatal("expected invalid trusted proxy CIDR to fail")
	}
}

func TestLoadRequiresPushLeaseLongerThanSendTimeout(t *testing.T) {
	t.Setenv("SSB_DATABASE_URL", "")
	t.Setenv("SSB_PUSH_LOCK_DURATION", "10s")
	t.Setenv("SSB_PUSH_SEND_TIMEOUT", "10s")

	if _, err := Load("healthcheck"); err == nil {
		t.Fatal("expected unsafe push lease to fail")
	}
}
