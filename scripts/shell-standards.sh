#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

readonly SUBCOMMAND="${1:?usage: shell-standards.sh <fmt|fmt-check|lint|syntax|policy|test>}"
readonly -a SHFMT_FLAGS=(-i 2 -ci -sr)

project_files() {
  if command -v git > /dev/null 2>&1 && git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    git ls-files -co --exclude-standard
  else
    find . \
      -type d \( -name .git -o -name bin -o -name coverage \
      -o -name dist -o -name out -o -name sbom -o -name tmp \
      -o -name vendor \) -prune \
      -o -type f -print | sed 's#^./##'
  fi
}

first_line() {
  local line=""
  IFS= read -r line < "$1" || :
  printf '%s' "${line}"
}

has_shebang() {
  [[ "$(first_line "$1")" == '#!'* ]]
}

shebang_dialect() {
  local command
  command="$(first_line "$1")"
  [[ "${command}" == '#!'* ]] || return 1
  command="${command#\#!}"

  case "${command}" in
    */env\ -S\ *) command="${command#*"/env -S "}" ;;
    */env\ *) command="${command#*"/env "}" ;;
    *) command="${command##*/}" ;;
  esac

  case "${command}" in
    bats | bats\ *)
      printf 'bats'
      ;;
    zsh | zsh\ *)
      printf 'zsh'
      ;;
    bash | bash\ *)
      printf 'bash'
      ;;
    sh | sh\ *)
      printf 'posix'
      ;;
    *)
      return 1
      ;;
  esac
}

has_recognized_shell_shebang() {
  shebang_dialect "$1" > /dev/null
}

is_shell_file() {
  case "$1" in
    *.sh | *.bash | *.bats | *.zsh)
      return 0
      ;;
    *) ;;
  esac

  has_recognized_shell_shebang "$1"
}

shell_files() {
  local file
  while IFS= read -r file; do
    [ -f "${file}" ] || continue
    if is_shell_file "${file}"; then
      printf '%s\n' "${file}"
    fi
  done < <(project_files)
}

dialect_for() {
  if shebang_dialect "$1"; then
    return
  fi

  case "$1" in
    *.bats)
      printf 'bats'
      ;;
    *.zsh)
      printf 'zsh'
      ;;
    *)
      printf 'bash'
      ;;
  esac
}

shellcheck_shell_for() {
  case "$1" in
    posix)
      printf 'sh'
      ;;
    *)
      printf '%s' "$1"
      ;;
  esac
}

is_glue_script() {
  case "$1" in
    scripts/* | bin/* | ci/* | tools/* | dev/*)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

policy_files() {
  local file
  while IFS= read -r file; do
    [ -f "${file}" ] || continue
    if is_shell_file "${file}" || { is_glue_script "${file}" && [ -x "${file}" ] && ! has_shebang "${file}"; }; then
      printf '%s\n' "${file}"
    fi
  done < <(project_files)
}

run_shfmt() {
  local mode="$1"
  local status=0
  local count=0
  local file dialect

  while IFS= read -r file; do
    dialect="$(dialect_for "${file}")"
    count=$((count + 1))

    if [ "${mode}" = "write" ]; then
      shfmt -w "${SHFMT_FLAGS[@]}" -ln "${dialect}" "${file}" || status=1
    else
      shfmt -d "${SHFMT_FLAGS[@]}" -ln "${dialect}" "${file}" || status=1
    fi
  done < <(shell_files)

  if [ "${count}" -eq 0 ]; then
    echo "No shell files found."
  fi

  return "${status}"
}

run_shellcheck() {
  local status=0
  local count=0
  local file dialect shell

  while IFS= read -r file; do
    dialect="$(dialect_for "${file}")"
    if [ "${dialect}" = "zsh" ]; then
      continue
    fi

    shell="$(shellcheck_shell_for "${dialect}")"
    count=$((count + 1))
    shellcheck -s "${shell}" "${file}" || status=1
  done < <(shell_files)

  if [ "${count}" -eq 0 ]; then
    echo "No ShellCheck-compatible shell files found."
  fi

  return "${status}"
}

run_syntax() {
  local status=0
  local count=0
  local file dialect

  while IFS= read -r file; do
    dialect="$(dialect_for "${file}")"
    case "${dialect}" in
      bash)
        count=$((count + 1))
        bash -n "${file}" || status=1
        ;;
      posix)
        count=$((count + 1))
        sh -n "${file}" || status=1
        ;;
      zsh)
        count=$((count + 1))
        if ! command -v zsh > /dev/null 2>&1; then
          echo "${file}: zsh is required to syntax-check zsh scripts." >&2
          status=1
        else
          zsh -n "${file}" || status=1
        fi
        ;;
      *) ;;
    esac
  done < <(shell_files)

  if [ "${count}" -eq 0 ]; then
    echo "No standalone shell scripts found for syntax checks."
  fi

  return "${status}"
}

run_policy() {
  local status=0
  local file

  while IFS= read -r file; do
    is_glue_script "${file}" || continue

    if ! has_recognized_shell_shebang "${file}"; then
      echo "${file}: project glue scripts need a recognized shell shebang." >&2
      status=1
    fi
  done < <(policy_files)

  return "${status}"
}

run_tests() {
  local files=()
  local count=0
  local file

  while IFS= read -r file; do
    case "${file}" in
      *.bats)
        files+=("${file}")
        count=$((count + 1))
        ;;
      *) ;;
    esac
  done < <(shell_files)

  if [ "${count}" -eq 0 ]; then
    echo "No Bats tests found."
    return 0
  fi

  bats "${files[@]}"
}

case "${SUBCOMMAND}" in
  fmt)
    run_shfmt write
    ;;
  fmt-check)
    run_shfmt check
    ;;
  lint)
    status=0
    run_shellcheck || status=1
    run_syntax || status=1
    run_policy || status=1
    exit "${status}"
    ;;
  syntax)
    run_syntax
    ;;
  policy)
    run_policy
    ;;
  test)
    run_tests
    ;;
  *)
    echo "Unknown shell standards subcommand: ${SUBCOMMAND}" >&2
    exit 2
    ;;
esac
