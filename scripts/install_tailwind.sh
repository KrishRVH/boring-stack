#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TAILWIND_VERSION="${TAILWIND_VERSION:?TAILWIND_VERSION must be set by mise.toml. Run: mise run tailwind}"
TAILWIND_LINUX_X64_SHA256="${TAILWIND_LINUX_X64_SHA256:-}"
TAILWIND_LINUX_ARM64_SHA256="${TAILWIND_LINUX_ARM64_SHA256:-}"

mkdir -p "$ROOT/bin"

os="$(uname -s)"
if [[ "$os" != "Linux" ]]; then
  echo "unsupported OS for the pinned standalone Tailwind binary: $os" >&2
  echo "This starter currently expects Linux or WSL. Set up an equivalent local Tailwind binary manually if needed." >&2
  exit 1
fi

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64)
    twarch="x64"
    expected_sha="$TAILWIND_LINUX_X64_SHA256"
    ;;
  aarch64 | arm64)
    twarch="arm64"
    expected_sha="$TAILWIND_LINUX_ARM64_SHA256"
    ;;
  *)
    echo "unsupported arch: $arch" >&2
    exit 1
    ;;
esac

url="https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-${twarch}"
out="$ROOT/bin/tailwindcss"
tmp="$(mktemp "${out}.XXXXXX")"
cleanup() {
  rm -f "$tmp"
}
trap cleanup EXIT

echo "Downloading Tailwind CSS ${TAILWIND_VERSION} standalone CLI: ${url}"
curl -fsSL -o "$tmp" "$url"
if [[ -n "$expected_sha" ]]; then
  actual_sha="$(sha256sum "$tmp" | awk '{print $1}')"
  if [[ "$actual_sha" != "$expected_sha" ]]; then
    echo "Tailwind CSS checksum mismatch for ${url}" >&2
    echo "  expected: $expected_sha" >&2
    echo "  actual:   $actual_sha" >&2
    exit 1
  fi
fi
chmod +x "$tmp"
"$tmp" --version > /dev/null 2>&1
mv "$tmp" "$out"
trap - EXIT
echo "Installed Tailwind CSS ${TAILWIND_VERSION} at ${out}"
