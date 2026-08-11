// Package lint holds the checks behind `goss lint`.
//
// Gossfiles are Go templates that render into YAML, which means nothing checks
// them today: yamllint can't read the unrendered form, and a bad function name
// only shows up when someone runs `goss validate` on a real server. These checks
// move that to authoring time.
package lint

import "sort"

// Severity is how much a finding matters. Warnings are reported but, on their
// own, don't change the exit code.
type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

func (s Severity) String() string {
	if s == SeverityError {
		return "error"
	}

	return "warning"
}

// Space records which file a finding's line number refers to.
//
// Rendering a gossfile moves lines: a `{{ if }}` that evaluates false takes its
// whole block out, and everything below shifts up. So a yamllint finding on line
// 40 of the rendered output is usually not line 40 of the file the user edits.
// Rather than report a line number that's quietly wrong, each finding says which
// file its number belongs to.
type Space int

const (
	// SpaceSource means the line number refers to the gossfile as written.
	SpaceSource Space = iota
	// SpaceRendered means it refers to the rendered YAML.
	SpaceRendered
)

// Rule identifiers. These are stable and documented in docs/lint.md, so
// findings can be looked up and suppressed by name.
const (
	RuleTemplateParse  = "template-parse"
	RuleDeprecatedFunc = "deprecated-func"
	RuleRender         = "render"
	RuleYAML           = "yaml"
)

// A Finding is one problem found in one gossfile.
type Finding struct {
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Col      int      `json:"col"`
	Rule     string   `json:"rule"`
	Message  string   `json:"message"`
	Severity Severity `json:"-"`
	Space    Space    `json:"-"`

	// Rendered is the path the rendered YAML was written to, set only on
	// findings in SpaceRendered so the reported line can actually be opened.
	Rendered string `json:"rendered,omitempty"`
}

// Sort orders findings the way someone reads a file: by path, then position,
// then rule name so the order is stable when two rules land on the same spot.
func Sort(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		switch {
		case a.File != b.File:
			return a.File < b.File
		case a.Line != b.Line:
			return a.Line < b.Line
		case a.Col != b.Col:
			return a.Col < b.Col
		default:
			return a.Rule < b.Rule
		}
	})
}

// HasErrors reports whether anything found is bad enough to fail a run.
func HasErrors(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityError {
			return true
		}
	}

	return false
}
