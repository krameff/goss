# Migration guide

## Coming from `goss-org/goss`

This project is a fork of [`goss-org/goss`](https://github.com/goss-org/goss), now
maintained at `github.com/krameff/goss`. The good news: your gossfiles don't need
to change. Nothing about the syntax, resource types, matchers, or CLI flags is
different — if it worked before, it still works.

What does change is where you get goss from:

* **Installing manually or via script?** Grab it from the
  [`krameff/goss` releases page](https://github.com/krameff/goss/releases), and use
  the [`install.sh`](https://github.com/krameff/goss/blob/master/install.sh) from this
  repo rather than `goss-org/goss` — see [Installation](installation.md) for details.
* **Using the container image?** Pull `ghcr.io/krameff/goss` instead of the old
  `aelsabbahy`/`goss-org` image.
* **Importing goss as a Go library?** Update your import path to
  `github.com/krameff/goss`.

That's it — everything else carries over as-is.

Worth knowing: this fork has also added a couple of features that don't
currently exist in upstream `goss-org/goss` (at time of writing) —
[discovery](gossfile.md#discovery) (run lightweight checks before the main
suite and feed the results into templates) and
[`depends-on`](gossfile.md#test-dependencies) (declare test prerequisites so
dependents are skipped, not failed, when a prerequisite fails). Neither is
required — existing gossfiles behave exactly as before — but they're there if
you want them.

## v4 migration

### Array matchers (e.g. user.groups) no longer allows duplicates

Goss v0.3.X allowed:

```yaml
user:
  root:
    exists: true
    groups:
      - root
      - root
      - root
```

Goss v0.4.x, will fail with the above as group "root" is only in the slice once. However, with goss v0.4.x the array may
contain matchers. The test below is valid for v0.4.x but not valid for v0.3.x

```yaml
user:
  root:
    exists: true
    groups:
      - have-prefix: r
```

## rpm now contains the full EVR version

To enable the ability to compare RPM versions in the future, The version matching of rpm has changed

from:

```console
rpm -q --nosignature --nohdrchk --nodigest --qf '%{VERSION}\n' package_name
```

to:

```console
rpm -q --nosignature --nohdrchk --nodigest --qf '%|EPOCH?{%{EPOCH}:}:{}|%{VERSION}-%{RELEASE}\n' package_name
```

## `file.contains` -> `file.contents`

File contains attribute has been renamed to file.contents

from:

```yaml
file:
  /tmp/foo:
    exists: true
    contains: []
```

to:

```yaml
file:
  /tmp/foo:
    exists: true
    contents: []
```
