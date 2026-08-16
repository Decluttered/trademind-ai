# Production Configuration Design (Phase P1)

## Goals

- One environment-variable contract without leaking development defaults into production.
- Fail-fast on insecure or missing production configuration.
- Redacted startup summary in logs.

## Load priority

From highest to lowest priority:

1. Process environment variables
2. Root `.env`
3. Safe, non-secret code defaults

`.env` is the only runtime file and `.env.example` is the only committed template. `APP_ENV` selects the profile.

## Profiles

| APP_ENV | Local storage | Demo seed | Dev routes |
| --- | --- | --- | --- |
| development | yes | default on | default on |
| demo | yes | default on | off |
| test | yes | off | off |
| staging | no | off | off |
| production | no | off | off |

## Fail-fast (production)

Blocking: missing DB, weak JWT, missing APP_MASTER_KEY, missing public URLs, demo seed / dev routes enabled.

Degraded (warn in config status): AI/OCR/Douyin Store credentials, storage public_base not E2E tested.

## Implementation

- `backend/internal/config/config.go` — env fields
- `backend/internal/config/validate.go` — production gates
- `backend/internal/config/summary.go` — log redaction

See also: [ENVIRONMENT_PROFILE_GUIDE.md](ENVIRONMENT_PROFILE_GUIDE.md), [SECRET_MANAGEMENT_GUIDE.md](SECRET_MANAGEMENT_GUIDE.md).
