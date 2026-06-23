# Changelog

### Security
- CLI announce output logs resource type and ID only; no longer marshals and prints full resource JSON (fixes CodeQL clear-text logging of password and other sensitive fields)
- Health probe debug logging records HTTP status only; no longer logs response body on non-OK status (avoids clear-text logging of password and other sensitive fields from test output)

### Fixed
- Integration test `add.goss.yaml` expectations updated for announce output change (resource type and ID only, not full marshaled YAML)
- `bullseye` apache2 version bumped to `2.4.67-1~deb11u3` in `vars.yaml` (Debian security update)
- Integration test fixtures: removed duplicate `service: apache2` / `service: httpd` definitions between `goss-shared.yaml` and `goss-service.yaml`; eliminates duplicate-key warnings during `validate` in CI

### Added
- Multiple `--vars` files supported; vars are merged in flag order with later files overriding overlapping keys; `--vars-inline` still applies last ([#1023](https://github.com/goss-org/goss/issues/1023); thanks to [@Lirt](https://github.com/Lirt) for [PR #1024](https://github.com/goss-org/goss/pull/1024))

### PRs Incorporated

- Thanks to [@kgaughan](https://github.com/kgaughan) for authoring the `urfave/cli` v3 migration
  - [#1060](https://github.com/goss-org/goss/pull/1060) - CLI migrated from `urfave/cli` v1 to `urfave/cli/v3` v3.9.0; v3 is actively maintained and drops transitive dependencies (`go-md2man`, `blackfriday`)
- Thanks to [@kgaughan](https://github.com/kgaughan) for restoring clearer `ContainElements` matcher error messages
  - [#1067](https://github.com/goss-org/goss/pull/1067) - pre-validates array/slice/map types before delegating to gomega, restoring the pre-iterator error text for invalid inputs (e.g. strings)
- Thanks to [@kgaughan](https://github.com/kgaughan) for the dependency and tooling refresh
  - [#1064](https://github.com/goss-org/goss/pull/1064) - golangci-lint v2.12.2 config (staticcheck settings, `noinlineerr` disabled), dependency bumps (`gomega` v1.41.0, `prometheus/common` v0.68.1), Trivy action updates, and minor lint cleanups; fork already had most workflow, Dockerfile, and code changes at equal or newer versions

- updated workflows
  - actions/checkout@v6.0.3 to 7.0.0
  - softprops/action-gh-release@v3.0.0 to softprops/action-gh-release@v3.0.1
  - Added permissions
---

## [0.5.0] - 2026-06-09

### Security
- `Dockerfile` final stage bumped from `alpine:3.19` to `alpine:3.21`; resolves CVE-2026-40200, CVE-2026-6042 (`musl-utils`) and CVE-2025-46394, CVE-2024-58251 (`ssl_client`/busybox)
- `Dockerfile` build stage `GO_VERSION` updated from `1.22` to `1.26`
- `SECURITY.md` added with vulnerability reporting policy and response timeline
- `CODE_OF_CONDUCT.md` added based on Contributor Covenant 2.1

### Added
- `linux/arm64` integration tests via native `ubuntu-24.04-arm` GitHub Actions runner
- `linux/ppc64le` integration tests via QEMU binfmt_misc emulation on `ubuntu-latest`
- `linux/ppc64le` binary added to release builds
- `integration-tests/goss/linux-arm64/` and `integration-tests/goss/linux-ppc64le/` test directories covering commands, addr, dns, file, group, kernel-param, http, and process resources
- `docker/setup-qemu-action` in CI to support transparent ppc64le binary execution without a container wrapper
- `.claude/agents/test-runner.md` agent for running integration tests in parallel

### PRs Incorporated

- [#1013](https://github.com/goss-org/goss/pull/1013) -- `http` resource: `request-query-params` added to support URL query parameters with proper encoding and duplicate key support; thanks to [@riton](https://github.com/riton)
- [#1017](https://github.com/goss-org/goss/pull/1017) -- `service` resource: Windows service checks implemented via PowerShell (`Get-Service`); supports `enabled`, `running`, and `exists`; `skip: true` removed from Windows integration test; thanks to [@cthiel42](https://github.com/cthiel42)
- [#1053](https://github.com/goss-org/goss/pull/1053) -- `registry` resource added for Windows registry validation; supports `exists`, `value`, and `type` checks; uses native `golang.org/x/sys/windows/registry` API (no PowerShell); five hives supported (HKLM, HKCU, HKCR, HKU, HKCC); six data types (REG_SZ, REG_EXPAND_SZ, REG_DWORD, REG_QWORD, REG_BINARY, REG_MULTI_SZ); `::` separator for value names containing backslashes; thanks to [@Blankf](https://github.com/Blankf)

### Changed
- `CODEOWNERS` updated to `@uk-bolly`
- `dependabot.yml` -- `docker` ecosystem added for Dockerfile base image tracking; assignees and reviewers added to all entries; `open-pull-requests-limit: 0` removed from gomod entry
- `ISSUE_TEMPLATE` -- all templates assigned to `uk-bolly`; `config.yml` added to redirect security reports to GitHub private vulnerability reporting
- `pull_request_template.md` -- `make test-all` corrected to `make test`
- `run-validate-tests.sh` Linux block narrowed to `linux-amd64` only; non-amd64 Linux architectures now testable via this path
- All platform command test files normalised: `--use-alpha=1` removed from exec commands (env var `GOSS_USE_ALPHA=1` is sufficient); `help.goss.yaml` stdout check unified to `validate` across all platforms
- Per-distro Makefile targets now depend on `release/goss-linux-amd64` only instead of the full `build` target

### Fixed
- `bullseye` apache2 updated to `2.4.67-1~deb11u2` in `vars.yaml` and expected files
- `generate_goss.sh` strips `::1` from generated localhost DNS entry; docker injects `::1` but podman does not, causing golden file mismatches in CI
- Service key order corrected for `rockylinux9` and `almalinux10`: `httpd` before `webservice` (alphabetical)
- `Dockerfile_jammy` and `Dockerfile_bullseye`: tinyproxy overridden to `Type=simple` with foreground mode; fixes startup failure in Docker CI due to `Type=forking` and `PrivateDevices=yes`

---

## [0.4.0] - Initial fork from goss-org/goss

### Documentation
- `README.md` updated to show origins, credits, and Apache 2.0 license retention

### Release pipeline
- `release.yaml` `TRAVIS_TAG` renamed to `RELEASE_TAG`
- `release.yaml` `attach-assets` job file glob corrected to match actual download paths
- `release-build.sh` fixed so `-p` flag correctly sets target platform, `os`, `arch`, and output filename
- `Makefile` release rule updated to pass `-p` and `-v` flags to `release-build.sh`

### CI
- `integration-test-linux-arm64` job added using native `ubuntu-24.04-arm` GitHub Actions runner; tests `goss-linux-arm64` binary via `run-validate-tests.sh` and `run-serve-tests.sh`
- `integration-test-linux-ppc64le` job added using QEMU binfmt_misc emulation on `ubuntu-latest`; `docker/setup-qemu-action` registers ppc64le handlers so the binary runs transparently without a container wrapper
- `run-validate-tests.sh` Linux block narrowed from all Linux to `linux-amd64` only; non-amd64 Linux architectures are now testable via this path
- `linux-arm64/` and `linux-ppc64le/` test directories added under `integration-tests/goss/`; cover commands, addr, dns, file, group, kernel-param, http, process, and gossfile resources
- `linux-arm64/commands/add.goss.yaml` and `linux-ppc64le/commands/add.goss.yaml` -- `add addr 127.0.0.1` (no port) marked `skip: true`; the format behaves differently on Linux vs Darwin
- All platform command test files (`darwin-amd64`, `darwin-arm64`, `linux-arm64`, `linux-ppc64le`, `windows`) normalised for consistency: `--use-alpha=1` removed from all `exec` commands (env var `GOSS_USE_ALPHA=1` set by `run-validate-tests.sh` is sufficient); `help.goss.yaml` stdout check changed from `alpha` to `validate` across all platforms
- `bullseye` apache2 version updated to `2.4.67-1~deb11u2` in `vars.yaml`, `goss-expected.yaml`, and `goss-aa-expected.yaml`
- `macos-13` (Intel) removed from CI matrix -- deprecated and no longer available on GitHub Actions; Apple Silicon testing continues via `macos-latest`
- `.travis.yml` removed; CI fully on GitHub Actions
- `docs.yaml` lint job re-enabled; build/deploy remains disabled
- `preview-docs.yaml` disabled
- `dependabot.yml` assignee and reviewer updated to `uk-bolly`
- `docker-integration-tests` workflow build context corrected to `integration-tests/`

### Security
- `Dockerfile` final stage bumped from `alpine:3.19` to `alpine:3.21`; resolves CVE-2026-40200 and CVE-2026-6042 (`musl-utils`) and CVE-2025-46394 and CVE-2024-58251 (`ssl_client`/busybox), all fixed in Alpine 3.21
- `Dockerfile` build stage `GO_VERSION` default updated from `1.22` to `1.26` to match `go.mod` requirement

### Build targets
- `linux/ppc64le` binary added to release builds
- `darwin/arm64` (Apple Silicon) binary added to release builds
- Removed 32-bit build and testing support

### Markdown lint fixes
- `README.md` attribution blockquote moved below heading (MD041); long lines wrapped (MD013); `[here]` link text made descriptive (MD059)
- `docs/gossfile.md` long admonition line wrapped (MD013); `[here]` link text made descriptive (MD059)
- `extras/dgoss/README.md` long line wrapped (MD013)
- `extras/kgoss/README.md` table pipe separators spaced correctly (MD060)

### Go and dependencies
- Updated to Go 1.26
- Replaced `github.com/achanda/go-sysctl` with `github.com/lorenzosaino/go-sysctl v0.3.1`
- Upgraded `github.com/BurntSushi/toml` v1.3.2 => v1.6.0
- Upgraded `golang.org/x/exp/typeparams`, `golang.org/x/lint`, `honnef.co/go/tools`
- GitHub Actions updated to later versions
- 386 build targets removed

### Linter
- `.golangci.yaml` migrated from v1 to v2 format
- CI updated to golangci-lint v2.12.2
- `go install` of golangci-lint removed from Makefile
- Error strings lowercased throughout to comply with ST1005
- Golden files updated for `TestMatchers` (`iter.Seq/iter.Seq2` support)
- `semver_constraint_test.go` assertions updated to match lowercased strings

### Integration tests
- `generate_goss.sh` strips `::1` from the generated localhost DNS entry after `goss add`; docker injects `::1` into `/etc/hosts` but podman does not, causing golden file mismatches
- `goss-expected.yaml` and `goss-expected-q.yaml` localhost DNS entries updated to remove `::1` across all distros to match the normalised generated output
- `goss-expected.yaml` and `goss-expected-q.yaml` service key order corrected for `rockylinux9` and `almalinux10`: `httpd` before `webservice` (alphabetical, matching goss output)
- `Dockerfile_jammy`: tinyproxy service overridden to `Type=simple` with foreground mode (`-d`), `ExecStartPre` creates `/run/tinyproxy`, and `PrivateDevices=no`; fixes tinyproxy failing to start in docker CI due to `Type=forking` PID file races and private device namespace restrictions
- `Dockerfile_jammy`: `/var/log/tinyproxy` pre-created with correct ownership; required by tinyproxy 1.11.0 on Ubuntu Jammy
- `Dockerfile_bullseye`: same tinyproxy service override applied as jammy; identical root cause
- `Makefile`: per-distro targets (`jammy`, `bullseye`, `rockylinux9`, etc.) now depend on `release/goss-linux-amd64` only instead of the full `build` target; avoids cross-compiling Windows and Darwin binaries for Linux integration tests
- `goss-expected.yaml` updated for `rockylinux9`, `almalinux10`, `bullseye`, `jammy`, and `alpine3` to include `::1` in `localhost` DNS addresses
- `goss-shared.yaml` User-Agent regex relaxed to match any goss version string, not just strict semver
- `goss-service.yaml` test service renamed from `foobar` to `webservice`; all distro `goss-expected.yaml` files and `generate_goss.sh` updated to match
- Redundant `bypath: goss-dummy.yaml` removed from all distro `goss.yaml` files; `goss-shared.yaml` is the single import point
- Thanks to [@kgaughan](https://github.com/kgaughan) for the integration test infrastructure overhaul
  ([PR #1061](https://github.com/goss-org/goss/pull/1061))
- External `dnstest.io` dependency replaced with a local dnsmasq zone on `127.0.0.1:8053`, making DNS tests self-contained
- Debian Bullseye and Ubuntu Jammy added with full test suites
- Ubuntu Trusty and Debian Wheezy removed (end of life); `.md5` sidecar files removed
- Alpine upgraded to 3.20; dnsmasq added
- Arch Linux dnsmasq and tinyproxy added
- RockyLinux 9 dnsmasq added
- `integration-test` CI job split into `integration-test-linux` and `integration-test-other`
- `integration-test-linux` further split into a per-distro matrix so each distro runs as an independent CI job
- AlmaLinux 10 integration test support added
- Integration test directories split by arch: `darwin/` renamed to `darwin-amd64/`; `darwin-arm64/` added
- CI matrix extended with `macos-13` (Intel) alongside `macos-latest` (Apple Silicon)
- `CentOS 7` yum redirected to `vault.centos.org` after EOL decommission
