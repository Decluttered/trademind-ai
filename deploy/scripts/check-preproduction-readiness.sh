#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ENV_FILE="$ROOT/.env"
source "$ROOT/deploy/scripts/load-env-defaults.sh"
load_env_defaults "$ENV_FILE"
node "$ROOT/scripts/preproduction-preflight.mjs" --mode startup >/dev/null

BASE_URL="http://127.0.0.1:${P10_BACKEND_BIND_PORT:-18080}"
BODY="$(mktemp)"
trap 'rm -f "$BODY"' EXIT

for attempt in $(seq 1 30); do
  code="$(curl -sS -o "$BODY" -w '%{http_code}' --max-time 5 "$BASE_URL/health/ready" || true)"
  if [[ "$code" == "200" ]] && node -e '
    const fs = require("node:fs");
    const payload = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
    const checks = payload.data?.checks || {};
    if (checks.database !== "ok" || checks.redis !== "ok" || checks.migrations !== "ok" || checks.appEnv !== "staging") process.exit(1);
  ' "$BODY"; then
    echo "[preproduction-readiness] database, redis, migrations, and staging profile are ready"
    exit 0
  fi
  if [[ "$attempt" -lt 30 ]]; then sleep 2; fi
done

echo "[preproduction-readiness] readiness failed without printing response content" >&2
exit 1
