package lint

import (
	"strings"
	"testing"
	"text/template"

	"github.com/go-sprout/sprout/sprigin"
)

// testFuncs mirrors how package goss composes its map: sprout first, then goss's
// own functions on top. The names here are goss's custom ones; the bodies don't
// matter, since nothing is executed during these checks.
func testFuncs() template.FuncMap {
	funcs := sprigin.TxtFuncMap()
	for _, name := range []string{
		"mkSlice", "readFile", "getEnv", "regexMatch",
		"toUpper", "toLower", "findStringSubmatch",
	} {
		funcs[name] = func(_ ...any) string { return "" }
	}

	return funcs
}

func TestDeprecatedFuncsComesFromSprout(t *testing.T) {
	got := DeprecatedFuncs()

	// If sprout ever stops exposing its notices, this rule would silently stop
	// finding anything. Fail loudly instead.
	if len(got) == 0 {
		t.Fatal("no deprecated functions found: sprout's Notices() API may have changed")
	}

	// Spot check a few we know are aliases of renamed functions.
	for _, name := range []string{"upper", "camelcase", "toYaml"} {
		if _, ok := got[name]; !ok {
			t.Errorf("expected %q in sprout's deprecated list", name)
		}
	}

	// Names goss overrides with its own implementations are not deprecated.
	if _, ok := got["toUpper"]; ok {
		t.Error("toUpper should not be reported as deprecated")
	}
}

func TestCheckTemplateParseErrors(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantLine int
		wantSub  string
	}{
		{
			name:     "unknown function",
			src:      "file:\n  /tmp:\n    title: {{ totallyBogus \"x\" }}\n",
			wantLine: 3,
			wantSub:  `function "totallyBogus" not defined`,
		},
		{
			name:     "unclosed action",
			src:      "file:\n  /tmp:\n    title: {{ toUpper \"x\"\n",
			wantLine: 4,
			wantSub:  "unclosed action",
		},
		{
			name:     "bad pipeline",
			src:      "a: {{ | toUpper }}\n",
			wantLine: 1,
			wantSub:  `unexpected "|"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckTemplate("goss.yaml", []byte(tc.src), testFuncs())

			if len(got) != 1 {
				t.Fatalf("expected 1 finding, got %d: %+v", len(got), got)
			}
			f := got[0]
			if f.Rule != RuleTemplateParse {
				t.Errorf("rule = %q, want %q", f.Rule, RuleTemplateParse)
			}
			if f.Severity != SeverityError {
				t.Errorf("severity = %v, want error", f.Severity)
			}
			if f.Space != SpaceSource {
				t.Errorf("space = %v, want source", f.Space)
			}
			if f.Line != tc.wantLine {
				t.Errorf("line = %d, want %d (message: %s)", f.Line, tc.wantLine, f.Message)
			}
			if !strings.Contains(f.Message, tc.wantSub) {
				t.Errorf("message = %q, want it to contain %q", f.Message, tc.wantSub)
			}
			// The "template: gossfile:N:" prefix is noise once the finding
			// carries file and line itself.
			if strings.HasPrefix(f.Message, "template:") {
				t.Errorf("message still carries the template prefix: %q", f.Message)
			}
		})
	}
}

func TestCheckTemplateDeprecatedFuncs(t *testing.T) {
	tests := []struct {
		name  string
		src   string
		want  []wantFinding
		clean bool
	}{
		{
			name: "simple pipeline",
			src:  "a: {{ \"hi\" | upper }}\n",
			want: []wantFinding{{1, 14, "upper"}},
		},
		{
			name: "several in one pipeline",
			src:  "a: {{ \"hi\" | upper | repeat 3 }}\n",
			want: []wantFinding{{1, 14, "upper"}},
		},
		{
			name: "inside an if",
			src:  "{{ if .Vars.x }}\na: {{ toYaml .Vars.y }}\n{{ end }}\n",
			want: []wantFinding{{2, 7, "toYaml"}},
		},
		{
			name: "inside a range",
			src:  "{{ range .Vars.list }}\na: {{ camelcase . }}\n{{ end }}\n",
			want: []wantFinding{{2, 7, "camelcase"}},
		},
		{
			name: "nested call as argument",
			src:  "a: {{ default \"x\" (upper \"y\") }}\n",
			want: []wantFinding{{1, 20, "upper"}},
		},
		{
			// A {{define}} body is parsed into its own template, so walking
			// only the root tree would miss this.
			name: "inside a define",
			src:  "{{ define \"foo\" }}{{ b64enc .X }}{{ end }}\n",
			want: []wantFinding{{1, 22, "b64enc"}},
		},
		{
			name: "inside a block",
			src:  "{{ block \"b\" . }}{{ upper .X }}{{ end }}\n",
			want: []wantFinding{{1, 21, "upper"}},
		},
		{
			// A parenthesized pipeline with a field access wraps the pipeline
			// in a ChainNode.
			name: "chained field access",
			src:  "a: {{ (toYaml .X).Field }}\n",
			want: []wantFinding{{1, 8, "toYaml"}},
		},
		{
			name: "assigned to a variable",
			src:  "{{ $v := upper .X }}a: {{ $v }}\n",
			want: []wantFinding{{1, 10, "upper"}},
		},
		{
			name:  "modern names only",
			src:   "a: {{ \"hi\" | toUpper }}\nb: {{ toYAML .Vars.x }}\n",
			clean: true,
		},
		{
			name:  "goss custom functions are not flagged",
			src:   "a: {{ mkSlice \"x\" }}\nb: {{ getEnv \"HOME\" \"/root\" }}\n",
			clean: true,
		},
		{
			name:  "no template at all",
			src:   "file:\n  /tmp:\n    exists: true\n",
			clean: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckTemplate("goss.yaml", []byte(tc.src), testFuncs())

			for _, f := range got {
				if f.Rule == RuleTemplateParse {
					t.Fatalf("unexpected parse error: %s", f.Message)
				}
			}

			if tc.clean {
				if len(got) != 0 {
					t.Fatalf("expected no findings, got %+v", got)
				}
				return
			}

			if len(got) != len(tc.want) {
				t.Fatalf("got %d findings, want %d: %+v", len(got), len(tc.want), got)
			}
			for i, want := range tc.want {
				f := got[i]
				if f.Line != want.line || f.Col != want.col {
					t.Errorf("finding %d at %d:%d, want %d:%d", i, f.Line, f.Col, want.line, want.col)
				}
				if !strings.Contains(f.Message, want.fn) {
					t.Errorf("finding %d message %q should name %q", i, f.Message, want.fn)
				}
			}
		})
	}
}

func TestCheckTemplateDeprecatedFuncSeverityAndSpace(t *testing.T) {
	got := CheckTemplate("goss.yaml", []byte("a: {{ \"hi\" | upper }}\n"), testFuncs())
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %+v", got)
	}

	f := got[0]
	if f.Severity != SeverityWarning {
		t.Errorf("severity = %v, want warning: deprecated names still work", f.Severity)
	}
	if f.Space != SpaceSource {
		t.Errorf("space = %v, want source", f.Space)
	}
	if !strings.Contains(f.Message, "toUpper") {
		t.Errorf("message should name the replacement, got %q", f.Message)
	}
}

func TestLineIndexPosition(t *testing.T) {
	src := []byte("abc\ndefgh\n\nij\n")
	idx := newLineIndex(src)

	tests := []struct {
		offset    int
		line, col int
	}{
		{0, 1, 1},
		{2, 1, 3},
		{3, 1, 4}, // the newline itself closes line 1
		{4, 2, 1}, // first byte of line 2
		{8, 2, 5},
		{10, 3, 1}, // empty line
		{11, 4, 1},
		{-1, 1, 1}, // defensive
	}

	for _, tc := range tests {
		line, col := idx.position(tc.offset)
		if line != tc.line || col != tc.col {
			t.Errorf("position(%d) = %d:%d, want %d:%d", tc.offset, line, col, tc.line, tc.col)
		}
	}
}

// wantFinding is the expected position and function name for one finding.
type wantFinding struct {
	line, col int
	fn        string
}
