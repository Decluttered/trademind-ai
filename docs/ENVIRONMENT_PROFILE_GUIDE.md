# Environment Profile Guide

## Supported profiles

Primary runtime profiles are `development`, `test`, `staging`, and `production`.
The existing `demo` and `performance` values remain specialized controlled modes; they do not have separate ENV files.

## Canonical files

- `.env` — the only runtime configuration file; keep it Git ignored.
- `.env.example` — the only committed configuration template; it contains no real secrets.

Copy `.env.example` to `.env`, then set `APP_ENV` and the values for that host. Do not create `.env.local`, `.env.test.local`, `.env.staging`, `.env.production`, or other ENV variants.

## Server deployment

1. Copy `.env.example` to the repository root as `.env`, or install an equivalent mode-`600` server file from the same canonical template.
2. Set `APP_ENV=production` and inject real secrets through the server runtime or secret manager.
3. Do **not** commit `.env`.

## Production dotenv rule

The API always considers only the repository-root `.env`; process environment variables take precedence. `APP_ENV` selects the profile, and secret values must not have code defaults.
