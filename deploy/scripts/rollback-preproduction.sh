#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ENV_FILE="$ROOT/.env"
source "$ROOT/deploy/scripts/load-env-defaults.sh"
load_env_defaults "$ENV_FILE"
COMPOSE=(docker compose --project-name trademind-preproduction --env-file "$ENV_FILE" -f "$ROOT/deploy/preproduction/compose.yml")
node "$ROOT/scripts/p10-preproduction-preflight.mjs" --mode rollback >/dev/null
export P10_API_IMAGE="$P10_PREVIOUS_API_IMAGE"
export P10_ADMIN_IMAGE="$P10_PREVIOUS_ADMIN_IMAGE"
"${COMPOSE[@]}" up -d backend admin
"$ROOT/deploy/scripts/check-preproduction-readiness.sh"
echo "[preproduction-rollback] application images rolled back; no database restore was executed"
