#!/usr/bin/env bash
# Runs `goss lint` over the gossfiles this repo ships, so our own examples and
# fixtures can't drift onto deprecated template functions or invalid YAML.
#
# Each file needs different vars to render, which is why this is a script rather
# than a one-line make target. Files that branch on a variable are linted with
# the branch taken and not taken, since a template only renders the side it
# takes and the other side would otherwise never be checked.
#
# Not strict by default: deprecated function names are warnings and don't fail
# the run. Set GOSS_LINT_STRICT=1 to fail on those too.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

GOSS_BIN="${GOSS_BIN:-${ROOT}/goss}"
LINT_ARGS=(lint --require-yamllint)
[[ "${GOSS_LINT_STRICT:-}" == "1" ]] && LINT_ARGS+=(--strict)

if [[ ! -x "${GOSS_BIN}" ]]; then
  echo "==> building goss"
  go build -o "${GOSS_BIN}" ./cmd/goss
fi

status=0

# lint <description> <goss global args...>
lint() {
  local desc="$1"
  shift

  echo "==> ${desc}"
  if ! "${GOSS_BIN}" "$@" "${LINT_ARGS[@]}"; then
    status=1
  fi
}

VARS="integration-tests/goss/vars.yaml"

# The shared and service gossfiles key their packages and services off $OS, so
# they need one of the distro names in vars.yaml to render at all. Which distro
# doesn't matter for linting, only that the lookups resolve.
export OS="${OS:-alpine3}"

lint "integration-tests/goss/goss-shared.yaml" \
  --vars "${VARS}" \
  --vars-inline '{"inline":"x","overwrite":"y"}' \
  -g integration-tests/goss/goss-shared.yaml

lint "integration-tests/goss/goss-service.yaml" \
  --vars "${VARS}" \
  -g integration-tests/goss/goss-service.yaml

lint "docs/goss.yaml" \
  --vars-inline '{"instance_count":1,"failures":0,"status":"OK"}' \
  -g docs/goss.yaml

# The discovery examples gate their whole body on a discovered value. `.Discovered`
# is fed from vars, so both branches can be rendered without running discovery.
for f in \
  integration-tests/goss/examples/discovery/goss.yml \
  integration-tests/goss/examples/discovery/goss-inline.yml \
  integration-tests/goss/examples/discovery/goss-with-deps.yml; do
  for found in true false; do
    lint "${f} (hosts_exists=${found})" \
      --vars-inline "{\"Discovered\":{\"hosts_exists\":${found}}}" \
      -g "${f}"
  done
done

if [[ ${status} -ne 0 ]]; then
  echo "gossfile lint: FAILED" >&2
  exit 1
fi

echo "gossfile lint: ok"
