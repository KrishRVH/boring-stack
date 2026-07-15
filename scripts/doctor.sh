#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$PROJECT_ROOT/scripts/lib/dev.sh"
cd "$PROJECT_ROOT"

mode="${1:-full}"
failures=0
warnings=0
docker_ready=0

mark_fail() {
  fail "$1"
  failures=$((failures + 1))
}

mark_warn() {
  warn "$1"
  warnings=$((warnings + 1))
}

check_mise() {
  if command -v mise > /dev/null 2>&1; then
    ok "mise is installed: $(mise --version | head -n 1)"
  else
    mark_fail "mise is missing. Install it with: curl https://mise.run | sh"
  fi
}

check_go() {
  local want have
  want="$(go_toolchain_version)"
  if ! command -v go > /dev/null 2>&1; then
    mark_fail "Go is missing from PATH. Run: mise install"
    return
  fi

  have="$(go env GOVERSION 2> /dev/null | sed 's/^go//')"
  if [[ "$have" == "$want" ]]; then
    ok "Go version matches go.mod toolchain: $have"
  else
    mark_fail "Go version is $have, expected $want from go.mod toolchain. Run: mise install"
  fi
}

check_goose_pin() {
  local want have
  want="${GOOSE_VERSION:-}"
  if [[ -z "$want" ]]; then
    return
  fi

  have="$(go list -m -f '{{.Version}}' github.com/pressly/goose/v3 2> /dev/null || true)"
  if [[ "$have" == "$want" ]]; then
    ok "Goose version matches go.mod: $have"
  else
    mark_fail "Goose version is $have in go.mod, expected $want from mise.toml. Update one pin so they match."
  fi
}

check_env_file() {
  if [[ -f .env ]]; then
    ok ".env exists"
  else
    mark_warn ".env is missing. Create it with: cp .env.example .env"
  fi
}

check_docker() {
  if require_docker; then
    ok "Docker and Docker Compose are usable"
    docker_ready=1
  else
    failures=$((failures + 1))
  fi
}

check_compose_config() {
  if [[ "$docker_ready" -ne 1 ]]; then
    return
  fi

  if docker compose config -q > /dev/null 2>&1; then
    ok "compose.yaml renders with mise environment"
  else
    mark_fail "compose.yaml did not render. Run this through mise so POSTGRES_IMAGE and NATS_IMAGE are set: mise run up"
  fi
}

check_compose_ports() {
  if [[ "$docker_ready" -ne 1 ]]; then
    return
  fi

  if require_free_compose_ports; then
    ok "Compose service ports are free or owned by this project"
  else
    failures=$((failures + 1))
  fi
}

check_port() {
  local port
  port="$(addr_port)"
  if [[ -z "$port" ]]; then
    mark_warn "ADDR has no obvious port: ${ADDR:-:8080}"
    return
  fi

  if port_in_use "$port"; then
    mark_warn "Port $port is already in use. Use ADDR=:8081 mise run dev, or stop the existing process."
  else
    ok "App port $port is free"
  fi
}

check_assets() {
  if [[ -x bin/tailwindcss ]]; then
    ok "Tailwind standalone binary exists"
  else
    mark_warn "Tailwind standalone binary is missing. Run: mise run tailwind"
  fi

  local missing_js=0
  local path
  for path in \
    web/assets/vendor/htmx.min.js \
    web/assets/vendor/sse.min.js \
    web/assets/vendor/alpine.min.js; do
    if [[ -s "$path" ]]; then
      ok "Browser asset exists: $path"
    else
      warn "Missing browser asset: $path"
      missing_js=1
    fi
  done

  if [[ "$missing_js" -ne 0 ]]; then
    mark_warn "Browser JS bundles are missing. Run: mise run vendor-js"
  elif [[ -f web/assets/vendor/checksums.sha256 ]]; then
    if (cd web/assets/vendor && sha256sum -c checksums.sha256 > /dev/null 2>&1); then
      ok "Browser asset checksums match"
    else
      mark_warn "Browser asset checksums do not match. Run: mise run vendor-js"
    fi
  else
    mark_warn "Browser asset checksums are missing. Run: mise run vendor-js"
  fi
}

check_generated_files_exist() {
  local missing=0
  local path
  for path in \
    internal/db/db.go \
    internal/db/models.go \
    internal/db/querier.go \
    internal/db/todos.sql.go \
    internal/ui/home_templ.go; do
    if [[ ! -s "$path" ]]; then
      warn "Missing generated file: $path"
      missing=1
    fi
  done

  if [[ "$missing" -eq 0 ]]; then
    ok "Checked-in generated files exist"
  else
    mark_fail "Generated files are missing. Run: mise run generate"
  fi
}

check_generated_drift() {
  if ! "$PROJECT_ROOT/scripts/check_generated_drift.sh"; then
    failures=$((failures + 1))
  fi
}

check_database() {
  if [[ "$docker_ready" -ne 1 ]]; then
    return
  fi

  if docker compose exec -T postgres pg_isready -U app -d app > /dev/null 2>&1; then
    ok "Postgres is reachable"
  else
    mark_warn "Postgres is not ready. Run: mise run wait-db, or inspect logs with: mise exec -- docker compose logs postgres"
  fi
}

printf 'Project doctor\n'
check_mise
check_go
check_goose_pin
check_env_file
check_docker
check_compose_config
check_compose_ports

if [[ "$mode" != "--quick" ]]; then
  check_port
  check_assets
  check_generated_files_exist
  check_generated_drift
  check_database
fi

if [[ "$failures" -gt 0 ]]; then
  printf '\nDoctor found %s failure(s) and %s warning(s).\n' "$failures" "$warnings" >&2
  exit 1
fi

if [[ "$warnings" -gt 0 ]]; then
  printf '\nDoctor passed with %s warning(s).\n' "$warnings"
else
  printf '\nDoctor passed.\n'
fi
