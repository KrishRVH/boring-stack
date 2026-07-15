#!/usr/bin/env bash

PROJECT_ROOT="${PROJECT_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"

ok() {
  printf '[ok] %s\n' "$*"
}

warn() {
  printf '[warn] %s\n' "$*" >&2
}

fail() {
  printf '[fail] %s\n' "$*" >&2
}

require_docker() {
  if ! command -v docker > /dev/null 2>&1; then
    fail "Docker is not installed or not on PATH."
    cat >&2 << 'EOF'
Fix:
  Install Docker Desktop with WSL integration enabled, or install Docker Engine inside Linux.
  Then confirm:
    docker version
    docker compose version
EOF
    return 1
  fi

  if ! docker version > /dev/null 2>&1; then
    fail "Docker is installed, but this shell cannot talk to the Docker daemon."
    cat >&2 << 'EOF'
Fix:
  Start Docker Desktop or the Docker service.
  On WSL, make sure Docker Desktop integration is enabled for this distro.
  If Docker Desktop reports a closed pipe while reading the Ubuntu home directory,
  close Docker Desktop, run `wsl.exe --shutdown` from PowerShell, reopen Ubuntu,
  start Docker Desktop, and confirm the distro is checked in:
    Docker Desktop > Settings > Resources > WSL integration
  Then confirm:
    docker version
EOF
    return 1
  fi

  if ! docker compose version > /dev/null 2>&1; then
    fail "Docker Compose v2 is not available."
    cat >&2 << 'EOF'
Fix:
  Install a recent Docker Desktop or Docker Engine package that includes `docker compose`.
  Then confirm:
    docker compose version
EOF
    return 1
  fi
}

go_toolchain_version() {
  local key value version=""

  while read -r key value _; do
    case "$key" in
      toolchain)
        printf '%s\n' "${value#go}"
        return 0
        ;;
      go)
        version="$value"
        ;;
      *) ;;
    esac
  done < "$PROJECT_ROOT/go.mod"

  if [[ -n "$version" ]]; then
    printf '%s\n' "$version"
    return 0
  fi

  return 1
}

addr_port() {
  local addr="${ADDR:-:8080}"
  case "$addr" in
    :*) printf '%s\n' "${addr#:}" ;;
    *:*) printf '%s\n' "${addr##*:}" ;;
    *) printf '%s\n' "" ;;
  esac
}

compose_port_postgres() {
  printf '%s\n' "${POSTGRES_PORT:-5432}"
}

compose_port_nats() {
  printf '%s\n' "${NATS_PORT:-4222}"
}

compose_port_nats_http() {
  printf '%s\n' "${NATS_HTTP_PORT:-8222}"
}

port_in_use() {
  local port="$1"
  if [[ -z "$port" ]]; then
    return 1
  fi

  if command -v ss > /dev/null 2>&1; then
    ss -ltn | awk '{print $4}' | grep -Eq "[:.]${port}$"
    return $?
  fi

  if command -v lsof > /dev/null 2>&1; then
    lsof -nP -iTCP:"$port" -sTCP:LISTEN > /dev/null 2>&1
    return $?
  fi

  return 1
}

port_owner() {
  local port="$1"
  local owner=""

  if command -v docker > /dev/null 2>&1; then
    owner="$(docker ps --format '{{.Names}} {{.Ports}}' 2> /dev/null | awk -v port="$port" '$0 ~ ":" port "->" {print}' | head -n 1 || true)"
    if [[ -n "$owner" ]]; then
      printf '%s\n' "$owner"
      return 0
    fi
  fi

  if command -v lsof > /dev/null 2>&1; then
    owner="$(lsof -nP -iTCP:"$port" -sTCP:LISTEN 2> /dev/null | awk 'NR==2 {print $1 " pid=" $2}' || true)"
    if [[ -n "$owner" ]]; then
      printf '%s\n' "$owner"
      return 0
    fi
  fi

  printf 'unknown process'
}

require_free_app_port() {
  local port
  port="$(addr_port)"
  if [[ -z "$port" ]]; then
    return 0
  fi

  if port_in_use "$port"; then
    fail "Port $port is already in use."
    warn "Owner: $(port_owner "$port")"
    cat >&2 << EOF
Fix:
  Stop the process using the port, or run the app on another port:
    ADDR=:8081 mise run dev
EOF
    return 1
  fi
}

compose_service_running() {
  local service="$1"
  docker compose ps --status running --services 2> /dev/null | grep -qx "$service"
}

require_free_compose_port() {
  local port="$1"
  local service="$2"
  local label="$3"

  if [[ -z "$port" ]]; then
    return 0
  fi

  if port_in_use "$port" && ! compose_service_running "$service"; then
    fail "Port $port is already in use, and it is not this project's $label container."
    warn "Owner: $(port_owner "$port")"
    return 1
  fi

  return 0
}

require_free_compose_ports() {
  local conflict=0

  if ! require_free_compose_port "$(compose_port_postgres)" postgres "Postgres"; then
    conflict=1
  fi
  if ! require_free_compose_port "$(compose_port_nats)" nats "NATS"; then
    conflict=1
  fi
  if ! require_free_compose_port "$(compose_port_nats_http)" nats "NATS"; then
    conflict=1
  fi

  if [[ "$conflict" -ne 0 ]]; then
    cat >&2 << 'EOF'
Fix:
  Stop the process using the port, or stop this project's containers:
    mise run down
  If another local project needs the defaults, choose alternate ports in .env:
    POSTGRES_PORT=5433
    NATS_PORT=4223
    NATS_HTTP_PORT=8223
  Then retry:
    mise run up
EOF
    return 1
  fi
}
