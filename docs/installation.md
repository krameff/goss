# Installation

For macOS and Windows platform notes, see [platforms](platforms.md).

## Build from source

```bash
make build
```

Platform binaries are written under `release/` (for example `release/goss-linux-amd64`).

Install a built binary on your `PATH`:

```bash
cp release/goss-linux-amd64 /usr/local/bin/goss   # adjust OS/arch as needed
chmod +rx /usr/local/bin/goss
```

Build a single platform target:

```bash
make release/goss-linux-amd64
```

Alternatively, build with [GoReleaser](https://goreleaser.com/):

```bash
goreleaser build --clean --single-target --snapshot
```

The binary is written under `dist/` (for example `dist/goss_linux_amd64_v1/goss`).

## dgoss and other wrappers

* [dgoss](../extras/dgoss/README.md) — run goss against Docker/Podman containers
* [kgoss](../extras/kgoss/README.md) — Kubernetes wrapper
* [dcgoss](../extras/dcgoss/README.md) — Docker Compose wrapper

## Release binaries

The supported install path is:

```bash
curl -fsSL https://goss.rocks/install | sh
```

Release assets are compressed archives named `goss_<version>_<os>_<arch>.tar.gz`
(for example `goss_0.5.0_linux_x86_64.tar.gz`). To install manually from a
GitHub release:

```bash
GOSS_VER=v0.5.0
curl -L "https://github.com/goss-org/goss/releases/download/${GOSS_VER}/goss_${GOSS_VER#v}_linux_x86_64.tar.gz" \
  | tar xz -C /tmp
sudo mv /tmp/goss /usr/local/bin/goss
chmod +rx /usr/local/bin/goss
```

Adjust the version, OS, and architecture in the filename as needed (`x86_64`,
`arm64`, `arm`, `s390x`, `i386`, etc.).

When release artifacts are published for this fork, download the matching archive
from the repository **Releases** page. Until then, use [build from source](#build-from-source) above.
