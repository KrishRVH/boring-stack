#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HTMX_VERSION="${HTMX_VERSION:?HTMX_VERSION must be set by mise.toml. Run: mise run vendor-js}"
HTMX_SSE_VERSION="${HTMX_SSE_VERSION:?HTMX_SSE_VERSION must be set by mise.toml. Run: mise run vendor-js}"
ALPINE_VERSION="${ALPINE_VERSION:?ALPINE_VERSION must be set by mise.toml. Run: mise run vendor-js}"

mkdir -p "$ROOT/web/assets/vendor"

download() {
  local name version url out
  name="$1"
  version="$2"
  url="$3"
  out="$4"

  echo "Downloading ${name} ${version}: ${url}"
  curl -fsSL -o "${out}.tmp" "$url"
  mv "${out}.tmp" "$out"
}

download "HTMX" "$HTMX_VERSION" \
  "https://cdn.jsdelivr.net/npm/htmx.org@${HTMX_VERSION}/dist/htmx.min.js" \
  "$ROOT/web/assets/vendor/htmx.min.js"

download "HTMX SSE extension" "$HTMX_SSE_VERSION" \
  "https://cdn.jsdelivr.net/npm/htmx-ext-sse@${HTMX_SSE_VERSION}/dist/sse.min.js" \
  "$ROOT/web/assets/vendor/sse.min.js"

download "Alpine" "$ALPINE_VERSION" \
  "https://cdn.jsdelivr.net/npm/alpinejs@${ALPINE_VERSION}/dist/cdn.min.js" \
  "$ROOT/web/assets/vendor/alpine.min.js"

(
  cd "$ROOT/web/assets/vendor"
  sha256sum \
    alpine.min.js \
    htmx.min.js \
    sse.min.js \
    > checksums.sha256
)
