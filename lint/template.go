package lint

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/sprigin"
)

// templateName is the name given to the parsed template. It shows up inside Go's
// parse error text, which is why the error rewriting below looks for it.
const templateName = "gossfile"

// parseErrRe pulls the position out of a text/template parse error, which
// formats as "template: <name>:<line>: <message>" and occasionally carries a
// column too.
var parseErrRe = regexp.MustCompile(`^template: ` + templateName + `:(\d+)(?::(\d+))?: (.*)$`)

// CheckTemplate runs the checks that work on the gossfile as written, before
// anything is rendered. Line numbers here refer to the source file.
//
// funcs must be the same function map goss renders with, otherwise every custom
// goss function looks undefined. Callers in package goss pass the real one.
//
// A parse failure stops the walk, because there's no tree to walk, and the parse
// error is the only finding worth reporting until it's fixed.
func CheckTemplate(file string, src []byte, funcs template.FuncMap) []Finding {
	tmpl, err := template.New(templateName).Funcs(funcs).Parse(string(src))
	if err != nil {
		return []Finding{parseFailure(file, err)}
	}

	return deprecatedFuncs(file, src, tmpl)
}

// parseFailure turns Go's parse error into a Finding, keeping the position it
// reports and dropping the "template: gossfile:12:" prefix, which is noise once
// the finding carries the file and line itself.
func parseFailure(file string, err error) Finding {
	f := Finding{
		File:     file,
		Line:     1,
		Rule:     RuleTemplateParse,
		Message:  err.Error(),
		Severity: SeverityError,
		Space:    SpaceSource,
	}

	m := parseErrRe.FindStringSubmatch(err.Error())
	if m == nil {
		return f
	}

	f.Line, _ = strconv.Atoi(m[1])
	if m[2] != "" {
		f.Col, _ = strconv.Atoi(m[2])
	}
	f.Message = m[3]

	return f
}

// execErrRe matches a text/template execution error, which carries the template
// name goss renders under rather than the one used for parsing.
var execErrRe = regexp.MustCompile(`^template: [^:]*:(\d+)(?::(\d+))?: (.*)$`)

// FromRenderError turns a failed render into a finding. Missing vars keys, bad
// pipelines and type errors all land here, with the position text/template
// reports, which refers to the source file since nothing has been substituted
// away yet.
func FromRenderError(file string, err error) Finding {
	f := Finding{
		File:     file,
		Line:     1,
		Rule:     RuleRender,
		Message:  err.Error(),
		Severity: SeverityError,
		Space:    SpaceSource,
	}

	m := execErrRe.FindStringSubmatch(err.Error())
	if m == nil {
		return f
	}

	f.Line, _ = strconv.Atoi(m[1])
	if m[2] != "" {
		f.Col, _ = strconv.Atoi(m[2])
	}
	// Drop the internal template name goss renders under. It means nothing to
	// someone reading their own gossfile.
	f.Message = strings.TrimPrefix(m[3], `executing "test" at `)

	return f
}

// deprecatedFuncs walks the parse tree looking for calls to function names
// sprout has deprecated. They all still work, which is the point: nothing else
// tells the user, so without this the gossfile silently rots.
func deprecatedFuncs(file string, src []byte, tmpl *template.Template) []Finding {
	deprecated := DeprecatedFuncs()
	if len(deprecated) == 0 || tmpl.Tree == nil {
		return nil
	}

	var findings []Finding
	lines := newLineIndex(src)

	// A {{define}} or {{block}} body is parsed into its own template, not into
	// the root tree, so walking Root alone would silently skip anything inside
	// one. Every tree here comes from the same Parse call, so the positions are
	// still offsets into src.
	for _, t := range tmpl.Templates() {
		if t.Tree == nil {
			continue
		}

		walk(t.Tree.Root, func(id *parse.IdentifierNode) {
			replacement, ok := deprecated[id.Ident]
			if !ok {
				return
			}

			line, col := lines.position(int(id.Position()))
			findings = append(findings, Finding{
				File:     file,
				Line:     line,
				Col:      col,
				Rule:     RuleDeprecatedFunc,
				Message:  fmt.Sprintf("%q is deprecated: %s", id.Ident, replacement),
				Severity: SeverityWarning,
				Space:    SpaceSource,
			})
		})
	}

	// Templates() iterates a map, so sort to keep the order stable.
	Sort(findings)

	return findings
}

// walk visits every node that can contain a function call. text/template has no
// generic visitor, so the node types that hold children are listed out.
func walk(n parse.Node, fn func(*parse.IdentifierNode)) {
	switch v := n.(type) {
	case nil:
		return
	case *parse.ListNode:
		if v == nil {
			return
		}
		for _, c := range v.Nodes {
			walk(c, fn)
		}
	case *parse.ActionNode:
		walk(v.Pipe, fn)
	case *parse.PipeNode:
		if v == nil {
			return
		}
		for _, c := range v.Cmds {
			walk(c, fn)
		}
	case *parse.CommandNode:
		for _, a := range v.Args {
			if id, ok := a.(*parse.IdentifierNode); ok {
				fn(id)
			}
			walk(a, fn)
		}
	case *parse.IfNode:
		walkBranch(v.BranchNode, fn)
	case *parse.RangeNode:
		walkBranch(v.BranchNode, fn)
	case *parse.WithNode:
		walkBranch(v.BranchNode, fn)
	case *parse.TemplateNode:
		walk(v.Pipe, fn)
	case *parse.ChainNode:
		// A parenthesized pipeline with a field access on the end, as in
		// {{ (toYaml .X).Field }}, wraps the pipeline in a chain.
		walk(v.Node, fn)
	}
}

func walkBranch(b parse.BranchNode, fn func(*parse.IdentifierNode)) {
	walk(b.Pipe, fn)
	walk(b.List, fn)
	walk(b.ElseList, fn)
}

// DeprecatedFuncs returns sprout's own list of deprecated template function
// names, mapped to the advice that comes with each one.
//
// Deriving this from sprout rather than keeping a copy means a sprout upgrade
// updates the rule by itself. There is a test asserting the list is non-empty,
// so if the upstream API ever stops producing it the rule fails loudly instead
// of quietly passing everything.
func DeprecatedFuncs() map[string]string {
	h := sprigin.NewSprigHandler()
	h.Build()

	out := make(map[string]string)
	for _, notice := range h.Notices() {
		if notice.Kind != sprout.NoticeKindDeprecated {
			continue
		}
		for _, name := range notice.FunctionNames {
			out[name] = strings.TrimSpace(notice.Message)
		}
	}

	return out
}

// lineIndex converts a byte offset into a line and column.
type lineIndex struct {
	starts []int
}

func newLineIndex(src []byte) *lineIndex {
	starts := []int{0}
	for i, b := range src {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}

	return &lineIndex{starts: starts}
}

// position returns a 1-based line and column for a byte offset.
func (l *lineIndex) position(offset int) (int, int) {
	if offset < 0 {
		return 1, 1
	}

	// The offset is past the start of the line we want, so walk back to the
	// last line that begins at or before it.
	line := 0
	for i, start := range l.starts {
		if start > offset {
			break
		}
		line = i
	}

	return line + 1, offset - l.starts[line] + 1
}
