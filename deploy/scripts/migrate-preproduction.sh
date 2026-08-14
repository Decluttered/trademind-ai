#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ENV_FILE="$ROOT/.env"
COMPOSE=(docker compose --project-name trademind-preproduction --env-file "$ENV_FILE" -f "$ROOT/deploy/preproduction/compose.yml")

node "$ROOT/scripts/preproduction-preflight.mjs" --mode migration
"${COMPOSE[@]}" up -d --wait postgres redis
"${COMPOSE[@]}" up -d backend
"$ROOT/deploy/scripts/check-preproduction-readiness.sh"
