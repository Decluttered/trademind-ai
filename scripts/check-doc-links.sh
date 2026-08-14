#!/usr/bin/env bash
# Public documentation consistency check wrapper.
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if command -v pwsh >/dev/null 2>&1; then
  pwsh -File "$REPO_ROOT/scripts/check-doc-links.ps1"
elif command -v powershell >/dev/null 2>&1; then
  powershell -ExecutionPolicy Bypass -File "$REPO_ROOT/scripts/check-doc-links.ps1"
else
  echo "PowerShell required for check-doc-links.ps1" >&2
  exit 1
fi
