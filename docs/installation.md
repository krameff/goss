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

## dgoss and other wrappers

* [dgoss](../extras/dgoss/README.md) — run goss against Docker/Podman containers
* [kgoss](../extras/kgoss/README.md) — Kubernetes wrapper
* [dcgoss](../extras/dcgoss/README.md) — Docker Compose wrapper

## Release binaries

When release artifacts are published for this fork, download the matching
`goss-<os>-<arch>` binary from the repository **Releases** page and install it on
your `PATH`. Until then, use [build from source](#build-from-source) above.
