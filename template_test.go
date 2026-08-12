package goss

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// render runs a template string through the standard (non-peek) template
// filter with no vars, and fails the test if rendering errors.
func render(t *testing.T, tpl string) string {
	t.Helper()

	filter, err := NewTemplateFilter([]string{}, `{}`, nil)
	if err != nil {
		t.Fatalf("template filter: %v", err)
	}

	out, err := filter([]byte(tpl))
	if err != nil {
		t.Fatalf("render %q: %v", tpl, err)
	}

	return string(out)
}

func assertRender(t *testing.T, tpl, want string) {
	t.Helper()

	if got := render(t, tpl); got != want {
		t.Errorf("%s\n got: %q\nwant: %q", tpl, got, want)
	}
}

// Goss registers its own funcMap after sprout's, so where the two libraries
// share a name, goss's implementation must win. See template.go.
func TestTemplateGossFuncsOverrideSprout(t *testing.T) {
	// sprout has toUpper/toLower too; goss's are plain strings.ToUpper/ToLower.
	assertRender(t, `{{ toUpper "aB c" }}`, "AB C")
	assertRender(t, `{{ toLower "aB C" }}`, "ab c")

	// sprout's env registry has no two-argument default form. This resolving
	// at all proves goss's getEnv is the one bound.
	t.Setenv("GOSS_TEMPLATE_TEST", "")
	assertRender(t, `{{ getEnv "GOSS_TEMPLATE_TEST" "fallback" }}`, "fallback")

	t.Setenv("GOSS_TEMPLATE_TEST", "set")
	assertRender(t, `{{ getEnv "GOSS_TEMPLATE_TEST" "fallback" }}`, "set")
}

// Goss's own template functions, previously uncovered by any Go test.
func TestTemplateGossFuncs(t *testing.T) {
	assertRender(t, `{{ index (mkSlice "x" "y" "z") 1 }}`, "y")
	assertRender(t, `{{ regexMatch "^ab" "abc" }}`, "true")
	assertRender(t, `{{ regexMatch "^zz" "abc" }}`, "false")

	dir := t.TempDir()
	path := filepath.Join(dir, "content")
	if err := os.WriteFile(path, []byte("  value\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	// readFile trims surrounding whitespace.
	assertRender(t, `{{ readFile "`+filepath.ToSlash(path)+`" }}`, "value")
}

// findStringSubmatch returns named subexpressions when the pattern has them,
// and stringified positional keys otherwise. Both forms are documented in
// docs/gossfile.md paired with sprout's `get`.
func TestTemplateFindStringSubmatch(t *testing.T) {
	assertRender(t, `{{ get (findStringSubmatch "(a)(b)" "ab") "1" }}`, "a")
	assertRender(t, `{{ get (findStringSubmatch "(a)(b)" "ab") "2" }}`, "b")
	assertRender(t, `{{ get (findStringSubmatch "(?P<first>a)(?P<second>b)" "ab") "first" }}`, "a")
	assertRender(t, `{{ get (findStringSubmatch "(?P<first>a)(?P<second>b)" "ab") "second" }}`, "b")
}

// Function names used in goss's shipped fixtures and documentation must keep
// resolving after the sprig -> sprout migration.
//
//	integration-tests/goss/goss-shared.yaml -- upper, repeat
//	docs/gossfile.md                        -- upper, repeat, get
func TestTemplateDocumentedSproutFuncs(t *testing.T) {
	assertRender(t, `{{ "hello!" | upper | repeat 5 }}`, strings.Repeat("HELLO!", 5))
	assertRender(t, `{{ default "fallback" "" }}`, "fallback")
	assertRender(t, `{{ trimSuffix "z" "abz" }}`, "ab")
	assertRender(t, `{{ quote "q" }}`, `"q"`)
	assertRender(t, `{{ toYaml (dict "a" 1) }}`, "a: 1")
}

// Sprout deliberately corrects a set of sprig behaviors rather than carrying
// the bugs forward (see SPRIG_TO_SPROUT_CHANGES_NOTES.md upstream). Pin the
// corrected values so a future sprout bump that regresses them is caught here
// instead of silently changing what users' gossfiles render.
func TestTemplateSproutBehaviorDeltasVsSprig(t *testing.T) {
	for _, tc := range []struct{ tpl, want, sprig string }{
		{`{{ "FoO  bar" | camelcase }}`, "FoOBar", "FoO Bar"},
		{`{{ camelcase "___complex__case_" }}`, "ComplexCase", "___Complex_Case_"},
		{`{{ "foo  bar" | kebabcase }}`, "foo-bar", "foo--bar"},
		{`{{ "foo  bar" | snakecase }}`, "foo_bar", "foo__bar"},
		{`{{ snakecase "Duration2m3s" }}`, "duration_2m_3s", "duration_2m3s"},
		{`{{ "foobar" | substr 0 -3 }}`, "foo", "foobar"},
		{`{{ "foobar" | substr -3 6 }}`, "bar", "foobar"},
		{`{{ "foooboooooo" | abbrevboth 4 9 }}`, "...boo...", "fooobo..."},
	} {
		if got := render(t, tc.tpl); got != tc.want {
			t.Errorf("%s\n got: %q\nwant: %q (sprig used to return %q)", tc.tpl, got, tc.want, tc.sprig)
		}
	}
}

// NewPeekTemplateFilter renders before discovery has run, so unknown
// .Discovered keys must evaluate as zero rather than failing the render.
func TestPeekTemplateFilterMissingDiscoveredKey(t *testing.T) {
	filter, err := NewPeekTemplateFilter([]string{}, `{}`)
	if err != nil {
		t.Fatalf("peek template filter: %v", err)
	}

	out, err := filter([]byte(`{{ if .Discovered.not_yet_known }}yes{{ else }}no{{ end }}`))
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if string(out) != "no" {
		t.Fatalf("unexpected rendered output: %q", string(out))
	}
}
