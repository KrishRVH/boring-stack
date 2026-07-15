#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$PROJECT_ROOT/scripts/lib/dev.sh"
cd "$PROJECT_ROOT"

require_docker

for _ in {1..60}; do
  if docker compose exec -T postgres pg_isready -U app -d app > /dev/null 2>&1; then
    echo "Postgres is ready."
    exit 0
  fi
  sleep 1
done

fail "Postgres did not become ready in 60 seconds."
cat >&2 << 'EOF'
Try:
  mise run up
  mise exec -- docker compose ps
  mise exec -- docker compose logs postgres

If the volume is from an incompatible Postgres version:
  mise run reset-db
EOF
exit 1
