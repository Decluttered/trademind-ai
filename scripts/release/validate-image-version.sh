#!/usr/bin/env bash
set -Eeuo pipefail

VERSION_FILE="${1:-deploy/IMAGE_VERSION}"

if [[ ! -f "$VERSION_FILE" ]]; then
  echo "::error::$VERSION_FILE is required" >&2
  exit 1
fi

version=''
{
  if ! IFS= read -r version && [[ -z "$version" ]]; then
    echo "::error::$VERSION_FILE must contain exactly one version line" >&2
    exit 1
  fi
  if IFS= read -r _; then
    echo "::error::$VERSION_FILE must contain exactly one version line" >&2
    exit 1
  fi
} < "$VERSION_FILE"

version="${version%$'\r'}"
identifier='(0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)'
semver_pattern="^(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-${identifier}(\\.${identifier})*)?$"

if (( ${#version} > 48 )) || [[ ! "$version" =~ $semver_pattern ]]; then
  echo "::error::$VERSION_FILE must contain a Docker-tag-safe SemVer of at most 48 characters without build metadata, such as 0.2.0 or 0.2.0-rc.1" >&2
  exit 1
fi

printf '%s\n' "$version"
