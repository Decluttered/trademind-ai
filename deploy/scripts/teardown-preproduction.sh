#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ENV_FILE="$ROOT/.env"
node "$ROOT/scripts/preproduction-preflight.mjs" --mode teardown >/dev/null
docker compose --project-name trademind-preproduction --env-file "$ENV_FILE" -f "$ROOT/deploy/preproduction/compose.yml" down
echo "[preproduction-teardown] services removed; isolated data volumes retained"
