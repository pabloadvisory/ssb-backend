# Production deployment

The production API is deployed from `main` to `germanyserver` at:

```text
https://ssb.ibuildnothing.com
```

The public API routes keep their existing paths, for example:

```text
https://ssb.ibuildnothing.com/health/ready
https://ssb.ibuildnothing.com/v1/leagues
https://ssb.ibuildnothing.com/v1/matches
https://ssb.ibuildnothing.com/v1/news
```

The mobile client base URL is `https://ssb.ibuildnothing.com`. Client code appends the `/v1/...` route.

## Server layout

```text
/srv/ssb-backend                         Git checkout of main
/srv/ssb-backend-data/assets             Persistent public assets, kept outside Git
/etc/ssb/backend.env                    root-only production secrets
/etc/cloudflared-ssb/config.yml         dedicated tunnel configuration
/etc/cloudflared-ssb/<tunnel-id>.json   dedicated tunnel credentials
/etc/systemd/system/cloudflared-ssb.service
127.0.0.1:8788                          private Docker origin
```

PostgreSQL is reachable only on the private Compose network. The API is reachable on the VPS only through loopback and is published through its dedicated Cloudflare Tunnel.

Team logos and other public files are stored outside the repository under `/srv/ssb-backend-data/assets`. The directory is mounted read-only into the API container and served at `https://ssb.ibuildnothing.com/assets/...`.

## Deploy a pushed change

After changes have passed tests and reached `origin/main`, run:

```bash
ssh root@germanyserver /srv/ssb-backend/deploy/deploy-production.sh
```

The script fast-forwards the server checkout, builds the production image, starts PostgreSQL, applies embedded migrations, replaces only the SSB API container, and waits for readiness. It stops without replacing the API if migrations fail.

## Verification

```bash
ssh root@germanyserver 'docker compose --env-file /etc/ssb/backend.env -f /srv/ssb-backend/deploy/docker-compose.production.yml ps'
ssh root@germanyserver 'curl -fsS http://127.0.0.1:8788/health/ready'
curl -fsS https://ssb.ibuildnothing.com/health/ready
curl -fsS 'https://ssb.ibuildnothing.com/v1/leagues?limit=20'
```

Do not commit `/etc/ssb/backend.env`, tunnel credential JSON, APNs keys, or Firebase service-account credentials.
