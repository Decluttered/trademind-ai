#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ENV_FILE="${P10_PREPRODUCTION_ENV_FILE:-$ROOT/.env.staging}"
node "$ROOT/scripts/p10-preproduction-preflight.mjs" --mode teardown --env-file "$ENV_FILE" >/dev/null
docker compose --project-name trademind-preproduction --env-file "$ENV_FILE" -f "$ROOT/deploy/preproduction/compose.yml" down
echo "[preproduction-teardown] services removed; isolated data volumes retained"
