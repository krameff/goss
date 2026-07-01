#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

TRIVY_SEVERITY="${TRIVY_SEVERITY:-HIGH,CRITICAL,MEDIUM}"
TRIVY_SKIP_DIRS="${TRIVY_SKIP_DIRS:-integration-tests,release,site,.venv,.git}"

run_govulncheck() {
  echo "==> govulncheck"
  if command -v govulncheck >/dev/null 2>&1; then
    govulncheck ./...
  else
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...
  fi
}

run_trivy() {
  trivy fs --scanners vuln --severity "${TRIVY_SEVERITY}" \
    --skip-dirs "${TRIVY_SKIP_DIRS}" \
    .
}

run_trivy_fs() {
  echo "==> trivy fs (go.mod, docs/requirements.txt)"

  if command -v trivy >/dev/null 2>&1; then
    run_trivy
    return
  fi

  if command -v docker >/dev/null 2>&1; then
    docker run --rm \
      -v "${ROOT}:/src" \
      -w /src \
      aquasec/trivy:latest \
      fs --scanners vuln --severity "${TRIVY_SEVERITY}" \
      --skip-dirs "${TRIVY_SKIP_DIRS}" \
      .
    return
  fi

  if [[ "${SECURITY_STRICT:-}" == "1" ]]; then
    echo "ERROR: trivy not found and docker unavailable (set SECURITY_STRICT=0 to skip)" >&2
    exit 1
  fi

  echo "WARN: skipping trivy fs scan (install trivy or run with docker available)" >&2
}

run_govulncheck
run_trivy_fs
