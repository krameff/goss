#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXAMPLES="${ROOT}/integration-tests/goss/examples/depends-on"
GOSS_ARGS=()

# shellcheck source=lib/goss-e2e-steps.sh
source "${ROOT}/ci/lib/goss-e2e-steps.sh"

if [[ -n "${GOSS_BINARY:-}" ]]; then
  GOSS="${GOSS_BINARY}"
elif [[ "$(uname -s)" == "Linux" ]]; then
  GOSS="${ROOT}/release/goss-linux-amd64"
  if [[ ! -x "${GOSS}" ]]; then
    make -C "${ROOT}" release/goss-linux-amd64
  fi
else
  GOSS="$(mktemp -t goss-depends-on-e2e.XXXXXX)"
  go build -o "${GOSS}" "${ROOT}/cmd/goss/goss.go"
  export GOSS_USE_ALPHA=1
  GOSS_ARGS=(--use-alpha=1)
fi

cleanup() {
  if [[ "${GOSS:-}" == /tmp/goss-depends-on-e2e.* ]] || [[ "${GOSS:-}" == *"/T/goss-depends-on-e2e."* ]]; then
    rm -f "${GOSS}"
  fi
}
trap cleanup EXIT

if [[ ! -x "${GOSS}" ]]; then
  echo "goss binary not found or not executable: ${GOSS}" >&2
  exit 1
fi

goss_runner() {
  "${GOSS}" "${GOSS_ARGS[@]}" "$@"
}

run_depends_on_e2e_steps "${EXAMPLES}" goss_runner

echo "depends-on e2e: ok"
