#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="${PROJECT_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
source "$PROJECT_ROOT/scripts/lib/dev.sh"
cd "$PROJECT_ROOT"

generated_manifest() {
  {
    find internal/db -maxdepth 1 -type f -name '*.go' ! -name doc.go -print
    find internal/ui -maxdepth 1 -type f -name '*_templ.go' -print
    printf '%s\n' web/assets/css/app.css
  } | sort
}

tmp="$(mktemp -d)"
manifest="$tmp/manifest"
new_manifest="$tmp/new-manifest"
log="$tmp/generate.log"
changed=0

restore_generated() {
  if [[ -f "$manifest" ]]; then
    if [[ -f "$new_manifest" ]]; then
      comm -13 "$manifest" "$new_manifest" | while IFS= read -r path; do
        rm -f "$path"
      done
    fi
    while IFS= read -r path; do
      if [[ -f "$tmp/original/$path" ]]; then
        cp "$tmp/original/$path" "$path"
      fi
    done < "$manifest"
  fi
  rm -rf "$tmp"
}
trap restore_generated EXIT

generated_manifest > "$manifest"
if [[ ! -s "$manifest" ]]; then
  fail "No generated files found. Run: mise run generate"
  exit 1
fi

while IFS= read -r path; do
  mkdir -p "$tmp/original/$(dirname "$path")"
  cp "$path" "$tmp/original/$path"
done < "$manifest"

if ! mise run regenerate > "$log" 2>&1; then
  sed 's/^/  /' "$log" >&2
  generated_manifest > "$new_manifest"
  fail "Regeneration failed. Run: mise run regenerate"
  exit 1
fi

generated_manifest > "$new_manifest"
if ! cmp -s "$manifest" "$new_manifest"; then
  changed=1
  warn "Generated file set changed:"
  comm -3 "$manifest" "$new_manifest" | sed 's/^/  /' >&2
fi

while IFS= read -r path; do
  if ! cmp -s "$tmp/original/$path" "$path"; then
    changed=1
    warn "Generated file is stale: $path"
  fi
done < "$manifest"

if [[ "$changed" -eq 0 ]]; then
  if [[ -f web/assets/vendor/checksums.sha256 ]]; then
    if ! (cd web/assets/vendor && sha256sum -c checksums.sha256) > "$tmp/vendor-checksums.log" 2>&1; then
      sed 's/^/  /' "$tmp/vendor-checksums.log" >&2
      fail "Vendored browser asset checksums failed. Run: mise run vendor-js"
      exit 1
    fi
  else
    fail "Missing web/assets/vendor/checksums.sha256. Run: mise run vendor-js"
    exit 1
  fi
  ok "Generated sqlc, templ, CSS, and vendored browser assets are current"
else
  fail "Generated files are stale. Run: mise run regenerate"
  exit 1
fi
