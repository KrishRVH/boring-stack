#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$PROJECT_ROOT/scripts/lib/dev.sh"
cd "$PROJECT_ROOT"

if ! ./scripts/wait_for_postgres.sh; then
  exit 1
fi

if ! go run ./cmd/migrate; then
  fail "Migrations failed."
  cat >&2 <<'EOF'
Check:
  mise exec -- docker compose ps
  mise exec -- docker compose logs postgres
  DB_URL in .env
EOF
  exit 1
fi
