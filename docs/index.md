# Goss - Quick and Easy server validation

<!-- markdownlint-disable no-inline-html -->
<p align="center">
  <img src="images/goss-logo.svg" alt="Goss - by Krameff Solutions Ltd" width="200">
</p>
<!-- markdownlint-enable no-inline-html -->

> ## This project is now Syver
>
> Goss has been renamed to **Syver** and all development has moved to
> **[github.com/krameff/syver](https://github.com/krameff/syver)**.
>
> This repository is frozen at **v0.6.0**. It will not get any further releases,
> fixes, or security updates, so please switch over to Syver.
>
> Your existing gossfiles carry across as they are. Syver still reads `goss.yaml`
> and the `gossfile:` key, so in most cases migrating just means installing the
> new binary and swapping `goss` for `syver` in whatever runs it.
>
> Why the rename: this fork has diverged a fair way from upstream
> [goss-org/goss](https://github.com/goss-org/goss), and sharing a binary name,
> config filename and release artifact names with it makes the two hard to tell
> apart. Its own name keeps them distinct, and keeps issues pointed at the right
> tracker.

--8<-- "README.md:intro"
--8<-- "README.md:about"

## Documentation

* [Installation](installation.md) — install goss, dgoss, and the other wrappers
* [Quickstart](quickstart.md) — write and run your first gossfile
* [Container image](container_image.md) — run goss from the published container image
* [Command reference](cli.md) — CLI flags for `validate`, `serve`, `add`, and friends
* [The gossfile](gossfile.md) — full resource and matcher reference, including
  [discovery](gossfile.md#discovery) and
  [test dependencies (`depends-on`)](gossfile.md#test-dependencies)
* [Migration guide](migrations.md) — breaking changes between versions
* [Platforms](platforms.md) — per-platform support notes and caveats
* Containers — [Docker](containers/docker.md), [Docker Compose](containers/docker-compose.md), [Kubernetes](containers/kubernetes.md)
* [Contributing](contributing.md) — development setup and contribution guidelines
* [Changelog](changelog.md) — release history
* [License](license.md)
