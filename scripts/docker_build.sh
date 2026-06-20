#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$PROJECT_ROOT/scripts/lib/dev.sh"
cd "$PROJECT_ROOT"

require_docker

go_version="$(go_toolchain_version)"

docker build \
  --build-arg GO_VERSION="$go_version" \
  --build-arg TEMPL_VERSION="${TEMPL_VERSION:?TEMPL_VERSION must be set by mise.toml}" \
  --build-arg SQLC_VERSION="${SQLC_VERSION:?SQLC_VERSION must be set by mise.toml}" \
  --build-arg GOOSE_VERSION="${GOOSE_VERSION:?GOOSE_VERSION must be set by mise.toml}" \
  --build-arg TAILWIND_VERSION="${TAILWIND_VERSION:?TAILWIND_VERSION must be set by mise.toml}" \
  --build-arg TAILWIND_LINUX_X64_SHA256="${TAILWIND_LINUX_X64_SHA256:-}" \
  --build-arg TAILWIND_LINUX_ARM64_SHA256="${TAILWIND_LINUX_ARM64_SHA256:-}" \
  --build-arg HTMX_VERSION="${HTMX_VERSION:?HTMX_VERSION must be set by mise.toml}" \
  --build-arg HTMX_SSE_VERSION="${HTMX_SSE_VERSION:?HTMX_SSE_VERSION must be set by mise.toml}" \
  --build-arg ALPINE_VERSION="${ALPINE_VERSION:?ALPINE_VERSION must be set by mise.toml}" \
  -t "${APP:?APP must be set by mise.toml}:local" .
