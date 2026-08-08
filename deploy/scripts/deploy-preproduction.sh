#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ENV_FILE="${P10_PREPRODUCTION_ENV_FILE:-$ROOT/.env.staging}"
export P10_PREPRODUCTION_ENV_FILE="$ENV_FILE"
COMPOSE=(docker compose --project-name trademind-preproduction --env-file "$ENV_FILE" -f "$ROOT/deploy/preproduction/compose.yml")

node "$ROOT/scripts/p10-preproduction-preflight.mjs" --mode startup --env-file "$ENV_FILE"
"${COMPOSE[@]}" config --quiet
"${COMPOSE[@]}" up -d --wait postgres redis
"${COMPOSE[@]}" up -d backend admin
P10_PREPRODUCTION_ENV_FILE="$ENV_FILE" "$ROOT/deploy/scripts/check-preproduction-readiness.sh"
