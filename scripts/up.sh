#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$PROJECT_ROOT/scripts/lib/dev.sh"
cd "$PROJECT_ROOT"

require_docker
require_free_compose_ports

if ! docker compose up -d postgres nats; then
  fail "Could not start Postgres and NATS."
  cat >&2 <<'EOF'
Try:
  mise exec -- docker compose ps
  mise exec -- docker compose logs postgres
  mise exec -- docker compose logs nats
EOF
  exit 1
fi

ok "Postgres and NATS are starting."
