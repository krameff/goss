package goss

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/krameff/goss/lint"
	"github.com/krameff/goss/util"
)

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
	// Strict makes warnings fail the run too. Deprecated function names are
	// warnings by default, so a repo can adopt the linter without having to
	// fix everything on day one.
	Strict bool
}

// Lint checks a gossfile and writes findings to out.
//
// The checks run in two stages. The template checks work on the file as
// written, so their line numbers match what the user is editing. The YAML
// checks need rendered output, so their line numbers refer to that instead;
// see lint.Space for why that distinction is kept rather than papered over.
func Lint(c *util.Config, opts LintOptions, out io.Writer) (int, error) {
	src, err := os.ReadFile(c.Spec)
	if err != nil {
		return LintError, err
	}

	findings := lint.CheckTemplate(c.Spec, src, TemplateFuncs())

	// Only render if the template parses. Rendering a file that failed to
	// parse just repeats the same error with less context.
	if !hasRule(findings, lint.RuleTemplateParse) {
		rendered, renderFindings, err := renderForLint(c, opts, src)
		if err != nil {
			return LintError, err
		}
		findings = append(findings, renderFindings...)

		if rendered != "" {
			if !lint.YamllintAvailable() && !opts.RequireYamllint {
				// Say so, otherwise a clean report looks like the YAML was
				// checked when it never was.
				fmt.Fprintln(os.Stderr, "yamllint not found on PATH, skipping YAML checks (use --require-yamllint to make this an error)")
			}

			yamlFindings, err := lint.Yamllint(c.Spec, rendered, lint.YamllintOptions{
				Config:   yamllintConfig(c, opts),
				Required: opts.RequireYamllint,
			})
			if err != nil {
				return LintError, err
			}
			findings = append(findings, yamlFindings...)
		}
	}

	if err := lint.Report(out, findings, opts.Format); err != nil {
		return LintError, err
	}

	return exitCode(findings, opts.Strict), nil
}

// renderForLint renders the gossfile and writes the result somewhere yamllint
// can read it. A render failure is a finding, not an error: it's a problem with
// the gossfile, which is exactly what the linter is for.
func renderForLint(c *util.Config, opts LintOptions, src []byte) (string, []lint.Finding, error) {
	filter, err := NewTemplateFilter(c.VarsFiles, c.VarsInline, nil)
	if err != nil {
		return "", nil, err
	}

	rendered, err := filter(src)
	if err != nil {
		return "", []lint.Finding{lint.FromRenderError(c.Spec, err)}, nil
	}

	path, err := writeRendered(c.Spec, opts.WriteRendered, rendered)
	if err != nil {
		return "", nil, err
	}

	return path, nil, nil
}

// writeRendered puts the rendered YAML where the reported line numbers can
// actually be opened. Without --write-rendered it goes to a temp file, which is
// left behind on purpose for the same reason.
func writeRendered(spec, dir string, rendered []byte) (string, error) {
	name := filepath.Base(spec)

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
func yamllintConfig(c *util.Config, opts LintOptions) string {
	if opts.YamllintConfig != "" {
		return opts.YamllintConfig
	}

	return lint.ConfigFor(c.Spec)
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
