#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$PROJECT_ROOT/scripts/lib/dev.sh"
cd "$PROJECT_ROOT"

require_docker
require_free_compose_ports
require_free_app_port

if [[ ! -f .env ]]; then
  cp .env.example .env
  ok "Created .env from .env.example"
fi

mise run setup
mise run up
mise run migrate

port="$(addr_port)"
printf '\nApp will start at http://localhost:%s\n\n' "${port:-8080}"
exec mise run dev
