#!/usr/bin/env bash
# Re-validates every entry in .trivyignore against a fresh, unfiltered trivy
# scan so suppressed findings don't go stale silently. Flags entries that:
#   - no longer appear in the scan (may be fixed/removed -- candidate to delete)
#   - now have a fixed version available (was blank before)
# Non-blocking by default: prints warnings but exits 0 unless
# TRIVYIGNORE_STRICT=1, since a missing trivy binary shouldn't block commits.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

IGNORE_FILE=".trivyignore"
TRIVY_SEVERITY="${TRIVY_SEVERITY:-HIGH,CRITICAL,MEDIUM,UNKNOWN}"
TRIVY_SKIP_DIRS="${TRIVY_SKIP_DIRS:-integration-tests,release,site,.venv,.git}"

if [[ ! -f "${IGNORE_FILE}" ]]; then
  echo "[trivyignore-check] no ${IGNORE_FILE}, skipping"
  exit 0
fi

mapfile -t ignored_ids < <(grep -vE '^\s*(#|$)' "${IGNORE_FILE}")

if [[ ${#ignored_ids[@]} -eq 0 ]]; then
  echo "[trivyignore-check] ${IGNORE_FILE} has no active entries, skipping"
  exit 0
fi

# --ignorefile /dev/null is essential: trivy picks up ./.trivyignore
# automatically, so without it the "fresh" scan below is filtered by the very
# file we're trying to validate and every entry looks like it disappeared.
run_trivy_json() {
  trivy fs --scanners vuln --severity "${TRIVY_SEVERITY}" \
    --skip-dirs "${TRIVY_SKIP_DIRS}" --ignorefile /dev/null \
    --format json --quiet .
}

run_trivy_json_container() {
  local engine="$1"
  "${engine}" run --rm -v "${ROOT}:/src" -w /src docker.io/aquasec/trivy:latest \
    fs --scanners vuln --severity "${TRIVY_SEVERITY}" \
    --skip-dirs "${TRIVY_SKIP_DIRS}" --ignorefile /dev/null \
    --format json --quiet .
}

scan_json=""
scan_failed=0

if command -v trivy >/dev/null 2>&1; then
  scan_json="$(run_trivy_json)" || scan_failed=1
elif command -v docker >/dev/null 2>&1; then
  scan_json="$(run_trivy_json_container docker)" || scan_failed=1
elif command -v podman >/dev/null 2>&1; then
  scan_json="$(run_trivy_json_container podman)" || scan_failed=1
else
  echo "WARN: [trivyignore-check] trivy not found and no container engine (docker/podman) available -- cannot re-validate ${IGNORE_FILE} entries: ${ignored_ids[*]}" >&2
  echo "WARN: install trivy or run with docker/podman available to verify these are still needed" >&2
  [[ "${TRIVYIGNORE_STRICT:-}" == "1" ]] && exit 1
  exit 0
fi

if [[ "${scan_failed}" -eq 1 || -z "${scan_json}" ]]; then
  echo "WARN: [trivyignore-check] trivy scan failed to run (e.g. vulnerability DB unreachable) -- cannot re-validate ${IGNORE_FILE} entries: ${ignored_ids[*]}" >&2
  echo "WARN: re-run manually once the trivy DB is reachable to verify these are still needed" >&2
  [[ "${TRIVYIGNORE_STRICT:-}" == "1" ]] && exit 1
  exit 0
fi

status=0
for id in "${ignored_ids[@]}"; do
  match="$(echo "${scan_json}" | jq -r --arg id "${id}" \
    '[.Results[]?.Vulnerabilities[]? | select(.VulnerabilityID == $id)] | .[0] // empty')"

  if [[ -z "${match}" ]]; then
    echo "WARN: [trivyignore-check] ${id} no longer found in scan results -- consider removing it from ${IGNORE_FILE}" >&2
    status=1
    continue
  fi

  fixed_version="$(echo "${match}" | jq -r '.FixedVersion // empty')"
  if [[ -n "${fixed_version}" ]]; then
    echo "WARN: [trivyignore-check] ${id} now has a fixed version (${fixed_version}) -- consider upgrading instead of suppressing" >&2
    status=1
  fi
done

if [[ ${status} -eq 0 ]]; then
  echo "[trivyignore-check] all ${#ignored_ids[@]} ${IGNORE_FILE} entries still apply, no fix available"
fi

[[ "${TRIVYIGNORE_STRICT:-}" == "1" ]] && exit "${status}"
exit 0
