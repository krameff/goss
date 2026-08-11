# Linting gossfiles

`goss lint` checks a gossfile while you are writing it, instead of leaving
problems to turn up when you run `goss validate` on a real server.

Gossfiles are awkward to check with normal tooling. They are Go templates that
render into YAML, so until they are rendered they are not valid YAML, and
yamllint cannot read them. `goss lint` handles both halves: it checks the
template as written, renders it, then hands the result to yamllint.

## Quick start

```console
$ goss -g goss.yaml lint
goss.yaml:8:14: warning: "upper" is deprecated: please use `toUpper` instead [deprecated-func]

0 errors, 1 warning
```

A clean file says so:

```console
$ goss -g goss.yaml lint
no problems found
```

If your gossfile uses variables, pass them the same way you would for
`validate` or `render`:

```console
goss --vars vars.yaml --vars-inline '{"env":"prod"}' -g goss.yaml lint
```

## What it checks

| Rule | Severity | What it catches |
| --- | --- | --- |
| `template-parse` | error | Template syntax errors and unknown function names |
| `deprecated-func` | warning | Template functions that still work but have been renamed |
| `render` | error | Failures while rendering, such as a missing vars key |
| `import` | error | A `gossfile:` import that matches no files |
| `yaml` | error or warning | Whatever yamllint reports about the rendered YAML |

### `template-parse`

The real template parser, so anything it rejects is genuinely broken. Most
often a typo in a function name.

```yaml
file:
  /tmp/example:
    title: {{ toUppr .Vars.name }}
```

```console
$ goss -g goss.yaml lint
goss.yaml:3: error: function "toUppr" not defined [template-parse]

1 error, 0 warnings
```

Fix the spelling:

```yaml
    title: {{ toUpper .Vars.name }}
```

When a file fails to parse, nothing else runs. There is no point rendering a
file that cannot be parsed, and the parse error is the thing to fix first.

### `deprecated-func`

goss gets its template functions from
[Sprout](https://docs.atom.codes/sprout). Sprout renamed a number of functions
it inherited from Sprig and kept the old names working as aliases. Those old
names still work in goss and are not going away, but the new names are clearer
and are what Sprout's own documentation uses.

```yaml
matching:
  greeting:
    content: {{ "hello!" | upper | repeat 5 }}
    matches:
      match-regexp: "HELLO!HELLO!HELLO!HELLO!HELLO!"
```

```console
$ goss -g goss.yaml lint
goss.yaml:3:28: warning: "upper" is deprecated: please use `toUpper` instead [deprecated-func]

0 errors, 1 warning
```

```yaml
    content: {{ "hello!" | toUpper | repeat 5 }}
```

This is a warning, not an error, so it will not fail a run on its own. Use
`--strict` if you want it to.

The list of deprecated names comes from Sprout itself at runtime rather than
being written down here, so it stays correct when Sprout is upgraded. Common
ones you are likely to hit: `upper`, `lower`, `title`, `camelcase`,
`snakecase`, `kebabcase`, `toYaml`, `toJson`, `b64enc`, `sha256sum`, `abbrev`.

Functions goss provides itself, such as `mkSlice`, `readFile`, `getEnv`,
`regexMatch`, `toUpper`, `toLower` and `findStringSubmatch`, are never reported.
They override the Sprout functions of the same name and are unaffected by
Sprout's renaming.

### `render`

Catches problems that only appear once the template runs, most often a variable
the gossfile expects and the vars file does not have.

```yaml
file:
  {{ .Vars.path }}:
    exists: true
```

```console
$ goss -g goss.yaml lint
goss.yaml:2:10: error: <.Vars.path>: map has no entry for key "path" [render]

1 error, 0 warnings
```

Either add `path` to your vars file, or pass it inline:

```console
$ goss --vars-inline '{"path":"/etc/hosts"}' -g goss.yaml lint
no problems found
```

### `import`

`goss lint` follows the `gossfile:` imports in the file you point it at, the
same way `validate` and `render` do, so one command covers a whole suite:

```console
goss --vars vars/CIS.yml -g standalone.yml lint
```

Globs are resolved relative to the importing file, entries marked `skip: true`
are left out, and a file already seen is not checked twice.

An import that matches nothing is reported, because goss refuses to run at all
in that case:

```console
$ goss -g goss.yml lint
goss.yml:1: error: import matches no files: section_4/cis_4.2.1/*.yml [import]
```

Usually a renamed or deleted directory that the entry point was never updated
for. Use `--no-imports` to check only the named file.

### `yaml`

Once the gossfile renders, the output goes to
[yamllint](https://yamllint.readthedocs.io/). This catches duplicate keys,
inconsistent indentation, bad spacing around colons, and anything that isn't
valid YAML at all.

```console
$ goss -g goss.yaml lint
goss.yaml:4:5 (rendered): error: duplication of key "exists" in mapping (key-duplicates) [yaml]

lines marked (rendered) refer to /tmp/goss-lint-3166187359/goss.yaml

1 error, 0 warnings
```

Note `(rendered)`. See [line numbers](#line-numbers) below.

yamllint is optional. If it is not installed, the YAML checks are skipped, the
other checks still run, and goss says so on stderr:

```console
$ goss -g goss.yaml lint
yamllint not found on PATH, skipping YAML checks (use --require-yamllint to make this an error)
no problems found
```

Install it with `pip install yamllint`. In CI, pass `--require-yamllint` so a
missing yamllint fails the build instead of quietly checking less than you
think.

#### Whitespace from template directives

goss does not report trailing spaces or blank lines in the rendered output, and
this is deliberate.

A control directive on its own line always leaves whitespace behind. The
directive produces no output, but the indentation in front of it and the newline
after it are not part of the directive and survive:

```yaml
gossfile:
  {{ if .Vars.section1 }}
  section_1/*/*.yml: {}
  {{ end }}
```

renders line 2 as two spaces and nothing else. Moving the directive to column 0
does not help, it just turns a trailing-space complaint into a blank-line one.
There is no way to write it that satisfies both rules.

Since the rendered file is a temporary intermediate that nobody reads, commits
or diffs, whitespace in it changes nothing. goss parses it identically either
way. So `trailing-spaces`, `empty-lines` and `new-line-at-end-of-file` are off
by default.

If you want the whitespace gone anyway, Go's trim markers do it: `{{-` removes
the whitespace before a tag, `-}}` the whitespace after. Worth knowing, not
worth restructuring an existing suite over.

#### yamllint configuration

Your own yamllint rules are used. goss looks for `.yamllint`, `.yamllint.yaml`
or `.yamllint.yml` starting in the gossfile's directory and walking up towards
the filesystem root, which finds the config at the top of most repositories.
Override the search with `--yamllint-config`.

If you have no config at all, goss uses yamllint's defaults with these turned
off:

* `trailing-spaces`, `empty-lines` and `new-line-at-end-of-file`, for the reason
  above: they describe a temporary file, and templating makes them unavoidable.
* `document-start`, because gossfiles do not begin with `---`, so leaving it on
  would warn about practically every file.
* `line-length`, because gossfile values are often long commands or paths that
  cannot be wrapped.

and one relaxed:

* `indent-sequences` is set to `consistent` rather than yamllint's default of
  requiring sequences to be indented under their key. Both styles are valid YAML
  and real gossfiles use both, so goss checks a file against whichever style it
  already uses. That still catches a file that mixes the two, without taking a
  side.

What remains is the structural half of yamllint: duplicate keys, inconsistent
indentation, colon spacing, and invalid YAML. Those can change what a gossfile
means, which is the point of checking.

Point `--yamllint-config` at your own file to take full control, including
turning any of these back on.

#### A recommended `.yamllint`

The built-in defaults are deliberately not a file, so nothing has to be created
to get started. If you want them written down, so your editor and any direct
`yamllint` runs agree with `goss lint`, put this at the top of your repository as
`.yamllint`:

```yaml
---
# yamllint config for a repository of gossfiles.
# Mirrors what `goss lint` uses when no config is found.
extends: default

rules:
  # A template directive on its own line always leaves whitespace behind once
  # rendered: indented it becomes a trailing space, at column 0 it becomes a
  # blank line. Neither changes what the gossfile means.
  trailing-spaces: disable
  empty-lines: disable
  new-line-at-end-of-file: disable

  # Gossfiles do not start with "---".
  document-start: disable

  # Values are often long commands, paths or regexes that cannot be wrapped.
  line-length: disable

  # Both sequence styles are valid YAML and gossfiles use both. "consistent"
  # still catches a file that mixes them, without picking a side.
  indentation:
    spaces: consistent
    indent-sequences: consistent
```

Once it exists, `goss lint` finds it by walking up from the gossfile and uses it
instead of the built-in defaults, so it is the one place to adjust the rules.

Note that running `yamllint` directly against your gossfiles still will not work
even with this config: they are templates, not YAML, until they are rendered.
That is what `goss lint` is for. If your repository also contains ordinary YAML
that you lint directly, add the gossfiles to an `ignore:` block so the two do
not fight:

```yaml
ignore:
  - goss.yaml
  - gossfiles/
```

## Fixing problems automatically

`--fix` rewrites your gossfiles to correct deprecated function names:

```console
$ goss -g goss.yaml lint --fix
no problems found

$ git diff
-    content: {{ "hello" | upper | repeat 2 }}
+    content: {{ "hello" | toUpper | repeat 2 }}
```

It follows imports like a normal run, so one command brings a whole suite up to
date.

### What it will not fix

Only deprecated names that are a straight rename. Everything else is reported
and left alone, deliberately.

**Anything reported against the rendered output**, which is all the `yaml`
findings: spacing, indentation, blank lines. There is no reliable way to map a
rendered line back to a line in your gossfile, since a `{{ if }}` that drops a
block shifts everything below it. Editing your source at a line number derived
from the rendered file would corrupt it. This is the opposite of what you might
expect: spacing looks like the easy thing to fix, and it is the one thing that
cannot be done safely.

**Deprecated functions whose call shape changed**, not just their name. `dig`
is the clear case: it moved from `{{ dig "key" "default" $dict }}` to
`{{ $dict | dig "key" | default "default" }}`. Swapping the identifier would
leave a file that still parses but renders differently, so it is reported for
you to change by hand.

**Parse errors, render failures and broken imports**, which all need a decision
about intent that a linter has no business making.

### The safety check

Before writing anything, goss renders the file both before and after the change
and compares the output byte for byte. A rename should be invisible in the
result. If it is not, the replacement was not equivalent, the edit is dropped,
and the finding is reported as normal with a note on stderr.

That said, `--fix` edits files in place. Run it on a clean working tree so you
can read the diff before committing.

## Line numbers

Findings come with a line number for one of two files, and each one says which.

Template findings point at the gossfile you wrote. Their line numbers are the
ones in your editor.

YAML findings point at the **rendered** output, and are marked `(rendered)`.
They have to be, because rendering moves lines around: a `{{ if }}` that
evaluates false takes its whole block out of the output, so everything below it
shifts up. Line 40 of the rendered YAML is often not line 40 of your gossfile.

Rather than report a number that is quietly wrong, goss tells you which file it
belongs to and writes that file out so you can open it. The path is printed at
the end of the report. To keep the rendered output somewhere predictable, use
`--write-rendered`:

```console
goss -g goss.yaml lint --write-rendered ./rendered
less ./rendered/goss.yaml
```

## Options

| Flag | Default | Purpose |
| --- | --- | --- |
| `--format` | `text` | `text`, `json` or `github` |
| `--strict` | off | Treat warnings as failures |
| `--no-imports` | off | Check only the named file, don't follow `gossfile:` imports |
| `--fix` | off | Rewrite gossfiles to correct what can be fixed safely |
| `--require-yamllint` | off | Fail if yamllint is not installed |
| `--yamllint-config` | search | Path to a yamllint config |
| `--write-rendered` | temp dir | Where to write the rendered gossfile |

The gossfile and variables come from goss's global options, so `-g`/`--gossfile`,
`--vars` and `--vars-inline` work exactly as they do for `validate` and
`render`.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | No problems, or warnings only without `--strict` |
| `1` | Problems found in the gossfile |
| `2` | The linter itself failed, for example a missing file or a bad `--format` |

`1` and `2` are kept separate on purpose, so CI can tell "your gossfile has a
problem" apart from "the linter is broken".

## Output formats

### `text`

The default, meant to be read by a person. Shown throughout this page.

### `json`

For anything that needs to consume the results.

```console
$ goss -g goss.yaml lint --format json
[
  {
    "file": "goss.yaml",
    "line": 8,
    "col": 14,
    "rule": "deprecated-func",
    "message": "\"upper\" is deprecated: please use `toUpper` instead"
  }
]
```

### `github`

Emits GitHub Actions annotations, so findings appear on the changed lines of a
pull request instead of being buried in the log.

```console
$ goss -g goss.yaml lint --format github
::warning file=goss.yaml,line=8,col=14::"upper" is deprecated: please use `toUpper` instead [deprecated-func]
```

## Using it in CI

### GitHub Actions

```yaml
name: lint gossfiles
on: [pull_request]

jobs:
  goss-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5

      - name: Install yamllint
        run: pip install yamllint

      - name: Install goss
        run: |
          curl -sSL https://github.com/krameff/goss/releases/latest/download/goss-linux-amd64 -o /usr/local/bin/goss
          chmod +x /usr/local/bin/goss

      - name: Lint gossfiles
        run: goss --vars vars.yaml -g goss.yaml lint --require-yamllint --format github
```

`--format github` puts each finding on the right line of the diff. Add
`--strict` when you are ready to treat deprecated names as failures.

### Several gossfiles

`goss lint` checks one gossfile per run, the same as `validate` and `render`.
Loop over them in the shell:

```bash
#!/usr/bin/env bash
set -euo pipefail

status=0
for f in gossfiles/*.yaml; do
  goss -g "$f" lint --require-yamllint --format github || status=1
done
exit $status
```

Collecting the failures rather than stopping at the first one means a single CI
run reports everything.

### Pre-commit hook

Check only what is being committed, so the hook stays quick:

```bash
#!/usr/bin/env bash
set -euo pipefail

staged=$(git diff --cached --name-only --diff-filter=ACM | grep -E 'goss.*\.ya?ml$' || true)
[ -z "$staged" ] && exit 0

for f in $staged; do
  goss -g "$f" lint || exit 1
done
```

### Makefile

Keeping it in a target means local runs and CI use the same command:

```make
GOSSFILES := $(wildcard gossfiles/*.yaml)

.PHONY: lint-goss
lint-goss:
	@for f in $(GOSSFILES); do goss -g $$f lint --require-yamllint || exit 1; done
```

## Limitations

It does not validate resource names or attributes. A misspelled resource type
passes the linter, and `goss validate` will not flag it either: the section is
simply not recognised, so its tests never run and you get `found 0 tests` if it
was the only one. Check the test count in the validate output rather than
relying on the linter for this.

Rendering needs whatever variables the gossfile references, so a file using
`.Vars` will report a `render` error unless you pass them. That is the same
requirement `goss render` has.
