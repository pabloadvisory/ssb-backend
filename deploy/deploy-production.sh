#!/bin/sh
set -eu

APP_DIR=${SSB_APP_DIR:-/srv/ssb-backend}
ENV_FILE=${SSB_ENV_FILE:-/etc/ssb/backend.env}
COMPOSE_FILE=${APP_DIR}/deploy/docker-compose.production.yml

if [ ! -d "${APP_DIR}/.git" ]; then
  echo "SSB repository is missing at ${APP_DIR}" >&2
  exit 1
fi

if [ ! -r "${ENV_FILE}" ]; then
  echo "SSB production environment is missing at ${ENV_FILE}" >&2
  exit 1
fi

cd "${APP_DIR}"

git pull --ff-only origin main

docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" build api
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" up -d postgres
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" run --rm migrate
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" up -d --remove-orphans api

attempt=1
while [ "${attempt}" -le 30 ]; do
  if curl --fail --silent --show-error http://127.0.0.1:8788/health/ready >/dev/null; then
    echo "SSB production API is ready on http://127.0.0.1:8788"
    exit 0
  fi
  sleep 2
  attempt=$((attempt + 1))
done

docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" ps >&2
docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" logs --tail=100 api >&2
echo "SSB production API did not become ready" >&2
exit 1
