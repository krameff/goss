package goss

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/krameff/goss/lint"
	"github.com/krameff/goss/util"
)

// maxLintImportDepth bounds the import walk, matching mergeJSONData.
const maxLintImportDepth = 50

// warnYamllintMissing keeps the "no yamllint" notice to once per run rather
// than once per imported file.
var warnYamllintMissing sync.Once

// Exit codes for `goss lint`. Kept distinct so CI can tell "your gossfile has
// problems" apart from "the linter itself broke".
const (
	LintOK       = 0
	LintFindings = 1
	LintError    = 2
)

// LintOptions carries the lint-specific flags. The gossfile and vars come from
// util.Config, since those are goss's normal global flags.
type LintOptions struct {
	Format          string
	YamllintConfig  string
	RequireYamllint bool
	WriteRendered   string
	// NoImports limits the run to the named gossfile instead of following
	// its `gossfile:` imports.
	NoImports bool
	// Fix rewrites the gossfile to correct what can be corrected safely.
	Fix bool
	// Strict makes warnings fail the run too. Deprecated function names are
	// warnings by default, so a repo can adopt the linter without having to
	// fix everything on day one.
	Strict bool
}

// Lint checks a gossfile and writes findings to out.
//
// By default it follows the `gossfile:` imports the same way `validate` and
// `render` do, so linting the entry point covers the whole suite. --no-imports
// limits it to the one file.
func Lint(c *util.Config, opts LintOptions, out io.Writer) (int, error) {
	// ReadJSON renders through this while resolving imports.
	var err error
	currentTemplateFilter, err = NewTemplateFilter(c.VarsFiles, c.VarsInline, nil)
	if err != nil {
		return LintError, err
	}

	targets, findings := lintTargets(c, opts)
	for _, spec := range targets {
		f, err := lintFile(spec, c, opts)
		if err != nil {
			return LintError, err
		}
		findings = append(findings, f...)
	}

	if err := lint.Report(out, findings, opts.Format); err != nil {
		return LintError, err
	}

	return exitCode(findings, opts.Strict), nil
}

// lintTargets is the entry gossfile plus everything it imports, unless the
// caller asked for just the one file.
func lintTargets(c *util.Config, opts LintOptions) ([]string, []lint.Finding) {
	if opts.NoImports {
		return []string{c.Spec}, nil
	}

	w := &importWalk{seen: map[string]bool{}}
	w.collect(c.Spec, 0)

	return w.targets, w.findings
}

// importWalk carries the state of one traversal of the import tree.
type importWalk struct {
	seen     map[string]bool
	targets  []string
	findings []lint.Finding
}

// collectGossfiles walks the import tree, mirroring how mergeJSONData resolves
// them: globs are relative to the importing file's directory, entries marked
// skip are left out, and the depth is bounded the same way.
//
// A file that can't be read or parsed is still returned, so the checks run on
// it and report why. Reporting a real finding beats failing the whole run.
func (w *importWalk) collect(spec string, depth int) {
	abs, err := filepath.Abs(spec)
	if err != nil || w.seen[abs] || depth >= maxLintImportDepth {
		return
	}
	w.seen[abs] = true
	w.targets = append(w.targets, spec)

	// ReadJSON parses using the package-level store format, so it has to be
	// set from this file's extension first. Without it every parse fails and
	// the import tree silently looks empty.
	var err2 error
	outStoreFormat, err2 = getStoreFormatFromFileName(spec)
	if err2 != nil {
		return
	}

	cfg, err := ReadJSON(spec)
	if err != nil {
		return
	}

	dir := filepath.Dir(spec)
	var patterns []string
	for _, g := range cfg.Gossfiles {
		if g.GetSkip() {
			continue
		}
		pattern := g.GetGossfile()
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(dir, pattern)
		}
		patterns = append(patterns, pattern)
	}
	// Glob order is already sorted, but the map above is not.
	sort.Strings(patterns)

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			// goss itself refuses to run at all when an import matches
			// nothing ("no matched files were found"), so this is a real
			// break rather than a tidiness point.
			w.findings = append(w.findings, lint.Finding{
				File:     spec,
				Line:     1,
				Rule:     lint.RuleImport,
				Message:  fmt.Sprintf("import matches no files: %s", pattern),
				Severity: lint.SeverityError,
				Space:    lint.SpaceSource,
			})

			continue
		}
		for _, match := range matches {
			w.collect(match, depth+1)
		}
	}
}

// lintFile runs every check against a single gossfile.
//
// The checks run in two stages. The template checks work on the file as
// written, so their line numbers match what the user is editing. The YAML
// checks need rendered output, so their line numbers refer to that instead;
// see lint.Space for why that distinction is kept rather than papered over.
func lintFile(spec string, c *util.Config, opts LintOptions) ([]lint.Finding, error) {
	src, err := os.ReadFile(spec)
	if err != nil {
		return nil, err
	}

	findings := lint.CheckTemplate(spec, src, TemplateFuncs())

	if opts.Fix {
		fixed, remaining, err := applyFixes(spec, c, src, findings)
		if err != nil {
			return nil, err
		}
		if fixed != nil {
			src = fixed
		}
		findings = remaining
	}

	// Only render if the template parses. Rendering a file that failed to
	// parse just repeats the same error with less context.
	if hasRule(findings, lint.RuleTemplateParse) {
		return findings, nil
	}

	rendered, path, renderFindings, err := renderForLint(spec, c, opts, src)
	if err != nil {
		return nil, err
	}
	findings = append(findings, renderFindings...)
	if path == "" {
		return findings, nil
	}

	// Structural YAML first. If the output doesn't parse, the schema check and
	// yamllint have nothing to work with and would only restate the same
	// problem in their own words.
	if yamlFindings := lint.CheckYAML(spec, path, rendered); len(yamlFindings) > 0 {
		return append(findings, yamlFindings...), nil
	}

	findings = append(findings, checkSchema(spec, path, rendered)...)

	if !lint.YamllintAvailable() && !opts.RequireYamllint {
		// Say so once, otherwise a clean report looks like the style rules
		// ran when they never did.
		warnYamllintMissing.Do(func() {
			fmt.Fprintln(os.Stderr, "yamllint not found on PATH, skipping YAML style checks (use --require-yamllint to make this an error)")
		})
	}

	yamlFindings, err := lint.Yamllint(spec, path, lint.YamllintOptions{
		Config:   yamllintConfig(spec, opts),
		Required: opts.RequireYamllint,
	})
	if err != nil {
		return nil, err
	}

	return append(findings, yamlFindings...), nil
}

// checkSchema decodes the rendered output into goss's own config types with
// unknown fields rejected, which catches a resource type or attribute that
// doesn't exist -- `fille:` for `file:`, `runing:` for `running:`.
//
// Deliberately no JSON Schema. docs/schema.yaml exists for editor completion
// and is maintained by hand, so checking against it would mean the linter
// disagrees with goss whenever the two drift. The structs below are what goss
// actually runs on, so they can't drift from it by definition, and a new
// resource attribute is covered the day it's added.
//
// The per-resource attribute check is goss's own (util.ValidateSections, called
// from each resource map's UnmarshalYAML). It has always run -- but only at
// validate time, on a real server. All this does is move it forward to
// authoring time, which is the whole point of the linter.
func checkSchema(spec, renderedPath string, rendered []byte) []lint.Finding {
	dec := yaml.NewDecoder(bytes.NewReader(rendered))
	dec.KnownFields(true)

	err := dec.Decode(NewGossConfig())
	if err == nil || lint.IsEmptyDocument(err) {
		return nil
	}

	// Reusable anchor blocks live at the top level and are not resource types.
	// See lint.AnchorKeys.
	anchors := lint.AnchorKeys(rendered)

	var out []lint.Finding
	for _, f := range lint.FromYAMLError(spec, renderedPath, lint.RuleSchema, err) {
		if key, ok := unknownResourceType(f.Message); ok {
			if anchors[key] {
				continue
			}
			f.Message = fmt.Sprintf("unknown resource type %q", key)
		}
		out = append(out, f)
	}

	return out
}

// unknownFieldRe matches go-yaml's message for a key with no matching struct
// field, which names the Go type: "field fille not found in type goss.GossConfig".
var unknownFieldRe = regexp.MustCompile(`^field (\S+) not found in type (\S+)$`)

// unknownResourceType returns the key from a top-level unknown-field message.
//
// Only the top level, since that is where resource types live and where the
// anchor exemption applies. Anything deeper is left with go-yaml's own wording,
// which at least says which type it was reading.
func unknownResourceType(msg string) (string, bool) {
	m := unknownFieldRe.FindStringSubmatch(msg)
	if m == nil || m[2] != "goss.GossConfig" {
		return "", false
	}

	return m[1], true
}

// applyFixes rewrites the gossfile for the findings that can be corrected
// without guessing, and returns the new content plus the findings that remain.
//
// Only deprecated names with a straight rename qualify. Whatever else is
// reported -- YAML spacing, indentation, parse errors -- is deliberately left
// alone. Spacing findings in particular come from the rendered output, and
// there is no reliable mapping from a rendered line back to a source line, so
// "fixing" one would mean editing a line chosen more or less at random.
//
// Nothing is written unless the file still renders to byte-identical output
// afterwards. A rename should be invisible in the result; if it isn't, the
// replacement wasn't equivalent and the edit is dropped.
func applyFixes(spec string, c *util.Config, src []byte, findings []lint.Finding) ([]byte, []lint.Finding, error) {
	updated, applied := lint.Fix(src, findings)
	if len(applied) == 0 {
		return nil, findings, nil
	}

	filter, err := NewTemplateFilter(c.VarsFiles, c.VarsInline, nil)
	if err != nil {
		return nil, nil, err
	}

	before, errBefore := filter(src)
	after, errAfter := filter(updated)
	if errBefore != nil || errAfter != nil || !bytes.Equal(before, after) {
		// Keep the findings so the user still hears about them, and say why
		// they weren't fixed rather than failing silently.
		fmt.Fprintf(os.Stderr, "%s: skipped --fix, the rename would have changed the rendered output\n", spec)

		return nil, findings, nil
	}

	if err := os.WriteFile(spec, updated, 0o644); err != nil {
		return nil, nil, err
	}

	// Re-check the rewritten file so the report describes what is on disk now.
	return updated, lint.CheckTemplate(spec, updated, TemplateFuncs()), nil
}

// renderForLint renders the gossfile and writes the result somewhere yamllint
// can read it, returning both the bytes and that path. A render failure is a
// finding, not an error: it's a problem with the gossfile, which is exactly
// what the linter is for.
func renderForLint(spec string, c *util.Config, opts LintOptions, src []byte) ([]byte, string, []lint.Finding, error) {
	filter, err := NewTemplateFilter(c.VarsFiles, c.VarsInline, nil)
	if err != nil {
		return nil, "", nil, err
	}

	rendered, err := filter(src)
	if err != nil {
		return nil, "", []lint.Finding{lint.FromRenderError(spec, err)}, nil
	}

	path, err := writeRendered(spec, opts.WriteRendered, rendered)
	if err != nil {
		return nil, "", nil, err
	}

	return rendered, path, nil, nil
}

// writeRendered puts the rendered YAML where the reported line numbers can
// actually be opened. Without --write-rendered it goes to a temp file, which is
// left behind on purpose for the same reason.
func writeRendered(spec, dir string, rendered []byte) (string, error) {
	name := filepath.Base(spec)

	// Following imports means many files can share a basename, so derive one
	// that can't collide when they're all written to the same directory.
	if dir != "" {
		if rel, err := filepath.Abs(spec); err == nil {
			name = strings.ReplaceAll(strings.TrimPrefix(rel, string(filepath.Separator)), string(filepath.Separator), "_")
		}
	}

	if dir == "" {
		tmp, err := os.MkdirTemp("", "goss-lint-")
		if err != nil {
			return "", err
		}
		dir = tmp
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, rendered, 0o644); err != nil {
		return "", err
	}

	return path, nil
}

// yamllintConfig prefers an explicit --yamllint-config, then looks for one near
// the gossfile.
func yamllintConfig(spec string, opts LintOptions) string {
	if opts.YamllintConfig != "" {
		return opts.YamllintConfig
	}

	return lint.ConfigFor(spec)
}

func hasRule(findings []lint.Finding, rule string) bool {
	for _, f := range findings {
		if f.Rule == rule {
			return true
		}
	}

	return false
}

func exitCode(findings []lint.Finding, strict bool) int {
	if lint.HasErrors(findings) {
		return LintFindings
	}
	if strict && len(findings) > 0 {
		return LintFindings
	}

	return LintOK
}
