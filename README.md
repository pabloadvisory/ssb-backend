# SSB Backend

Production-oriented Go backend foundation for football data and editorial coverage: competitions, seasons, teams, venues, players, coaches, fixtures, live matches, lineups, events, standings, stories, announcements, and match reports.

Repository: [github.com/pabloadvisory/ssb-backend](https://github.com/pabloadvisory/ssb-backend). The Go module path matches it exactly.

## What exists today

- public, versioned read API for leagues, teams, players, coaches, matches, and match events;
- public, reverse-chronological news feed and article detail API with scheduled publication, filters, and football-resource relationships;
- isolated draft/published/archived editorial workflow with idempotent, authenticated CMS upserts;
- keyset pagination for stable high-volume match and league feeds;
- authenticated, idempotent match snapshot ingestion;
- provider-neutral PostgreSQL schema with external identity mappings, constraints, and indexes for the complete first football model;
- atomic, versioned outbox records for every match change, with asynchronous policy/fanout processing;
- cross-replica live updates using PostgreSQL `LISTEN/NOTIFY`, exposed over both SSE and WebSockets for iOS, Android, and web clients;
- anonymous installation credentials, rotatable iOS APNs/ActivityKit and Android FCM endpoints, and match notification subscriptions;
- a durable, lease-owner-tokened `SKIP LOCKED` delivery queue with bounded APNs + FCM concurrency, retry/backoff, token invalidation, and per-match collapse keys;
- per-client installation throttling, aggregate SSE/WebSocket connection caps, and explicit trusted-proxy CIDRs;
- bounded retention for terminal deliveries, completed/failed outbox events, and endpoint-transfer audit records;
- public conditional GETs with strong ETags and CDN-ready cache policy;
- queue depth/oldest-age, HTTP, worker, and realtime metrics behind a vendor-neutral adapter;
- liveness/readiness probes, structured JSON logs, request IDs, panic recovery, body limits, timeouts, awaited graceful shutdown, and non-root containers;
- embedded, checksummed, advisory-lock-protected migrations;
- unit and HTTP contract tests plus real-PostgreSQL migration, ordering, versioning, and lease tests in CI.

This is deliberately a modular monolith. It can scale horizontally and keeps transaction boundaries simple while the product and provider contracts are still evolving. Split ingestion, fanout, or analytics into services only when measurements justify it.

## Quick start

```bash
cp .env.example .env
set -a; source .env; set +a
docker compose up --build
curl http://localhost:8080/health/ready
curl 'http://localhost:8080/v1/leagues?limit=20'
curl 'http://localhost:8080/v1/news?limit=20'
```

In another terminal, load deterministic development fixtures:

```bash
make seed-demo
```

Or run PostgreSQL with Docker and the service locally:

```bash
docker compose up -d postgres
cp .env.example .env
set -a; source .env; set +a
go run ./cmd/ssb migrate up
go run ./cmd/ssb serve
```

## Commands

```text
ssb serve                     run the API, outbox publisher, live listener, and retention worker
ssb migrate up                apply pending embedded migrations
ssb seed demo                 reset idempotent demo fixtures (disabled in production)
ssb healthcheck [URL]         container-compatible readiness probe
ssb push-worker               deliver queued APNs/ActivityKit and Android FCM messages
```

## Demo data

`make seed-demo` creates one league, one current season, one venue, four teams, four players, two coaches, one official, standings with home/away splits, lineups/team formations, player match statistics, three matches (live, scheduled, and finished), two published news articles, and one private draft. Running it again resets the same fixed-ID fixtures without creating duplicates. The command refuses to run when `SSB_ENV=production`.

Useful fixture IDs:

```text
League:         10000000-0000-0000-0000-000000000001
Live match:     40000000-0000-0000-0000-000000000001
Scheduled:      40000000-0000-0000-0000-000000000002
Finished:       40000000-0000-0000-0000-000000000003
Team:           20000000-0000-0000-0000-000000000001
Player:         30000000-0000-0000-0000-000000000001
Coach:          30000000-0000-0000-0000-000000000005
News article:   50000000-0000-0000-0000-000000000001
```

Examples:

```bash
curl 'http://localhost:8080/v1/matches?status=live'
curl 'http://localhost:8080/v1/teams/20000000-0000-0000-0000-000000000001'
curl 'http://localhost:8080/v1/players/30000000-0000-0000-0000-000000000001'
curl 'http://localhost:8080/v1/coaches/30000000-0000-0000-0000-000000000005'
curl 'http://localhost:8080/v1/news?featured=true'
curl 'http://localhost:8080/v1/news/victoria-edge-mahe-in-demo-opener'
curl 'http://localhost:8080/v1/matches/40000000-0000-0000-0000-000000000001/events'
curl -N 'http://localhost:8080/v1/matches/40000000-0000-0000-0000-000000000001/stream'
```

## API baseline

```text
GET  /health/live
GET  /health/ready
GET  /v1/leagues
GET  /v1/leagues/{id}
GET  /v1/teams/{id}
GET  /v1/players/{id}
GET  /v1/coaches/{id}
GET  /v1/matches
GET  /v1/matches/{id}
GET  /v1/matches/{id}/events
GET  /v1/matches/{id}/stream
GET  /v1/matches/{id}/ws
GET  /v1/news
GET  /v1/news/{slug}
POST /v1/installations
PUT  /v1/installations/{id}/push-endpoints/{kind}
PUT  /v1/installations/{id}/matches/{match_id}
PUT  /v1/internal/matches/{provider}/{external_id}
PUT  /v1/internal/news/{source}/{external_id}
```

The internal endpoint requires `Authorization: Bearer <SSB_INGEST_API_KEY>`. Repeating the same provider/external ID updates the same match. Match versions increase only when the snapshot changes.

The internal news endpoint uses a separate `Authorization: Bearer <SSB_EDITORIAL_API_KEY>` credential. Leave that setting empty to disable editorial writes. Repeating the same source/external ID is idempotent: the canonical article ID and version remain stable until content changes.

`POST /v1/installations` returns an installation credential exactly once. Store it in the iOS Keychain or Android Keystore-backed encrypted storage. The two installation `PUT` routes require that credential as a bearer token. The API applies a bounded per-client token bucket to installation creation and caps aggregate SSE/WebSocket connections per client IP. Keep distributed limits at the production edge as defense in depth, and add App Attest / Play Integrity verification before public launch.

Endpoint kinds are:

- `standard`: APNs on iOS or FCM on Android for score/status notifications;
- `live_activity`: an ActivityKit token tied to one match; the client must re-register when `pushTokenUpdates` changes it;
- `push_to_start`: an ActivityKit push-to-start token, used when a subscribed match transitions to live.

The Live Activity payload contract assumes a Swift `FootballMatchAttributes` type with `matchID`, `homeTeamName`, and `awayTeamName`, and a `ContentState` with `homeScore`, `awayScore`, `status`, `elapsedMinute`, and `period`. Override the type name with `SSB_ACTIVITY_ATTRIBUTES_TYPE`.

Example match query:

```bash
curl 'http://localhost:8080/v1/matches?status=live&league_id=<uuid>&limit=50'
```

The response `page.next_cursor` is opaque. Pass it back as `cursor`; do not parse or construct it in clients.

Public JSON GETs return a strong `ETag` and `Cache-Control: public`. Send `If-None-Match` to receive `304 Not Modified` when a representation has not changed.

## News contract

The feed returns lightweight summaries in reverse publication order; article bodies are fetched only when a story opens:

```bash
curl 'http://localhost:8080/v1/news?limit=20&category=match_report&featured=true'
curl 'http://localhost:8080/v1/news?league_id=<uuid>&team_id=<uuid>&match_id=<uuid>'
curl 'http://localhost:8080/v1/news/{slug}'
```

Supported categories are `story`, `match_report`, and `announcement`. Optional `league_id`, `team_id`, and `match_id` fields let the app navigate from a story to its football context. The body field is `body_markdown`. Drafts, archived articles, and published articles with a future `published_at` never appear in public reads.

Editorial systems send a complete representation with `PUT /v1/internal/news/{source}/{external_id}`. For example:

```json
{
  "slug": "victoria-edge-mahe",
  "title": "Victoria edge Mahé",
  "summary": "Victoria United took all three points.",
  "body_markdown": "## Match report\n\nVictoria United...",
  "author_name": "SSB Newsroom",
  "category": "match_report",
  "featured": true,
  "related_league_id": "10000000-0000-0000-0000-000000000001",
  "related_match_id": "40000000-0000-0000-0000-000000000001",
  "status": "published",
  "published_at": "2026-08-01T10:00:00Z"
}
```

`status` accepts `draft`, `published`, or `archived`. A published article requires an explicit RFC 3339 `published_at`; use a future value for scheduled publication. Hero images remain object-storage/CDN URLs and can include accessible `hero_image_alt` text.

## Verification

```bash
make test                 # fast local tests; PostgreSQL integration test skips without its URL
make test-integration     # complete suite against the Compose PostgreSQL service
make vet
make staticcheck          # pinned Staticcheck 2026.1 / v0.7.0
make vuln
```

CI starts PostgreSQL 17, applies all forward migrations twice to prove idempotency, loads demo data, and exercises hash-gated match versions, per-aggregate outbox ordering, delivery lease reclaim/retry, and isolated push-token storage.

## Production notes

- Run migrations as a separate release job before starting new API instances.
- Put the API behind TLS and a gateway/load balancer. Apply distributed public limits there; keep the ingest route on a private network as well as using its key.
- Set `SSB_TRUSTED_PROXY_CIDRS` only to the load balancer/proxy networks that overwrite `X-Forwarded-For`. Forwarded headers from all other peers are ignored.
- `SSB_DATABASE_URL` must currently point directly at PostgreSQL because every API replica reserves one `LISTEN/NOTIFY` connection. Add a separate pooled read/write URL before introducing PgBouncer transaction mode.
- Set `SSB_INGEST_API_KEY` from a secret manager. Startup fails in production if it is missing or too short.
- Set `SSB_EDITORIAL_API_KEY` from a separate secret when editorial writes are enabled. Production rejects a configured key shorter than 32 characters.
- Run `ssb push-worker` separately. Configure APNs with `SSB_APNS_KEY_PATH`, `SSB_APNS_KEY_ID`, and `SSB_APNS_TEAM_ID`; configure Android with `SSB_FCM_PROJECT_ID` plus Google Application Default Credentials (`GOOGLE_APPLICATION_CREDENTIALS` off Google Cloud).
- APNs and FCM tokens are bearer capabilities. Raw values are isolated in `push_endpoint_tokens` so normal endpoint queries do not read them. Restrict that table separately and add application-level envelope encryption before handling real production tokens.
- PostgreSQL notifications are ephemeral and intentionally contain only IDs/version data. The transactional outbox consumer performs realtime publication and notification fanout after ingest commits; per-match ordering and idempotency protect multi-replica processing.
- Retention runs in bounded batches in API replicas. Defaults retain terminal deliveries for 30 days, completed/failed outbox events for 7 days, and endpoint-transfer audits for one year.
- Queue health is sampled every 30 seconds by default (`SSB_QUEUE_HEALTH_INTERVAL`) and emitted through the metrics adapter with queue depth and oldest-pending age.
- `SSB_PUSH_LOCK_DURATION` and `SSB_OUTBOX_LOCK_DURATION` must exceed their processing timeout by at least five seconds; startup rejects unsafe combinations.
- Pin container images by digest in the deployment repository. Compose uses convenient development tags.
- Object media should live in object storage/CDN; only URLs belong in this database.

Migrations are intentionally forward-only. Corrective schema changes must be added as a new checked migration rather than relying on untested rollback files.

See [docs/architecture.md](docs/architecture.md) for boundaries, scaling decisions, and the reference-repository review.
