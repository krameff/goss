package goss

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krameff/goss/lint"
	"github.com/krameff/goss/util"
)

// writeGossfiles lays out a small suite in a temp dir and returns its path.
func writeGossfiles(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func runLint(t *testing.T, spec string, opts LintOptions) (int, string) {
	t.Helper()

	if opts.Format == "" {
		opts.Format = lint.FormatText
	}

	var out bytes.Buffer
	cfg, err := util.NewConfig(util.WithSpecFile(spec))
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	code, err := Lint(cfg, opts, &out)
	if err != nil {
		t.Fatalf("lint: %v", err)
	}

	return code, out.String()
}

// Linting the entry point should cover everything it imports, so a problem in
// an imported file is still reported.
func TestLintFollowsGossfileImports(t *testing.T) {
	dir := writeGossfiles(t, map[string]string{
		"goss.yaml":  "gossfile:\n  sub/*.yaml: {}\n",
		"sub/a.yaml": "file:\n  /tmp:\n    title: {{ \"x\" | upper }}\n    exists: true\n",
		"sub/b.yaml": "file:\n  /var:\n    exists: true\n",
	})

	_, out := runLint(t, filepath.Join(dir, "goss.yaml"), LintOptions{})

	if !strings.Contains(out, "a.yaml") || !strings.Contains(out, "deprecated") {
		t.Errorf("expected the deprecated name in the imported file to be reported, got:\n%s", out)
	}
}

func TestLintNoImports(t *testing.T) {
	dir := writeGossfiles(t, map[string]string{
		"goss.yaml":  "gossfile:\n  sub/*.yaml: {}\n",
		"sub/a.yaml": "file:\n  /tmp:\n    title: {{ \"x\" | upper }}\n    exists: true\n",
	})

	_, out := runLint(t, filepath.Join(dir, "goss.yaml"), LintOptions{NoImports: true})

	if strings.Contains(out, "a.yaml") {
		t.Errorf("--no-imports should not reach imported files, got:\n%s", out)
	}
}

// An import marked skip is not followed, matching how validate treats it.
func TestLintSkipsSkippedImports(t *testing.T) {
	dir := writeGossfiles(t, map[string]string{
		"goss.yaml":  "gossfile:\n  sub/a.yaml:\n    skip: true\n",
		"sub/a.yaml": "file:\n  /tmp:\n    title: {{ \"x\" | upper }}\n    exists: true\n",
	})

	_, out := runLint(t, filepath.Join(dir, "goss.yaml"), LintOptions{})

	if strings.Contains(out, "a.yaml") {
		t.Errorf("a skipped import should not be linted, got:\n%s", out)
	}
}

// goss refuses to run when an import matches nothing, so the linter says so.
func TestLintReportsUnmatchedImport(t *testing.T) {
	dir := writeGossfiles(t, map[string]string{
		"goss.yaml": "gossfile:\n  missing/*.yaml: {}\n",
	})

	code, out := runLint(t, filepath.Join(dir, "goss.yaml"), LintOptions{})

	if !strings.Contains(out, "import matches no files") {
		t.Errorf("expected an unmatched import to be reported, got:\n%s", out)
	}
	if code != LintFindings {
		t.Errorf("exit code = %d, want %d", code, LintFindings)
	}
}

// A cycle between two gossfiles must not hang or repeat.
func TestLintImportCycle(t *testing.T) {
	dir := writeGossfiles(t, map[string]string{
		"goss.yaml":  "gossfile:\n  other.yaml: {}\n",
		"other.yaml": "gossfile:\n  goss.yaml: {}\n",
	})

	code, out := runLint(t, filepath.Join(dir, "goss.yaml"), LintOptions{})

	if code == LintError {
		t.Errorf("a cycle should not fail the run, got:\n%s", out)
	}
}

func TestLintExitCodes(t *testing.T) {
	dir := writeGossfiles(t, map[string]string{
		"clean.yaml":      "file:\n  /tmp:\n    exists: true\n",
		"deprecated.yaml": "file:\n  /tmp:\n    title: {{ \"x\" | upper }}\n    exists: true\n",
		"broken.yaml":     "file:\n  /tmp:\n    title: {{ bogusFunc \"x\" }}\n",
	})

	tests := []struct {
		file   string
		strict bool
		want   int
	}{
		{"clean.yaml", false, LintOK},
		{"deprecated.yaml", false, LintOK},      // warnings alone don't fail
		{"deprecated.yaml", true, LintFindings}, // ...unless asked
		{"broken.yaml", false, LintFindings},
	}

	for _, tc := range tests {
		code, out := runLint(t, filepath.Join(dir, tc.file), LintOptions{Strict: tc.strict})
		if code != tc.want {
			t.Errorf("%s (strict=%v): exit %d, want %d\n%s", tc.file, tc.strict, code, tc.want, out)
		}
	}
}

// --fix rewrites the file on disk, and only for what it can do safely.
func TestLintFixRewritesFile(t *testing.T) {
	dir := writeGossfiles(t, map[string]string{
		"goss.yaml": "file:\n  /tmp:\n    title: {{ \"x\" | upper }}\n    exists: true\n",
	})
	spec := filepath.Join(dir, "goss.yaml")

	code, out := runLint(t, spec, LintOptions{Fix: true})

	body, err := os.ReadFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "toUpper") {
		t.Errorf("file was not rewritten:\n%s", body)
	}
	if strings.Contains(string(body), "| upper ") {
		t.Errorf("old name still present:\n%s", body)
	}
	if code != LintOK {
		t.Errorf("exit %d, want %d (the warning was fixed)\n%s", code, LintOK, out)
	}
}

// --fix follows imports too, so a whole suite can be brought up to date.
func TestLintFixAcrossImports(t *testing.T) {
	dir := writeGossfiles(t, map[string]string{
		"goss.yaml":  "gossfile:\n  sub/*.yaml: {}\n",
		"sub/a.yaml": "file:\n  /tmp:\n    title: {{ \"x\" | upper }}\n    exists: true\n",
	})

	runLint(t, filepath.Join(dir, "goss.yaml"), LintOptions{Fix: true})

	body, err := os.ReadFile(filepath.Join(dir, "sub", "a.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "toUpper") {
		t.Errorf("imported file was not fixed:\n%s", body)
	}
}

// Without --fix nothing on disk changes.
func TestLintDoesNotWriteWithoutFix(t *testing.T) {
	body := "file:\n  /tmp:\n    title: {{ \"x\" | upper }}\n    exists: true\n"
	dir := writeGossfiles(t, map[string]string{"goss.yaml": body})
	spec := filepath.Join(dir, "goss.yaml")

	runLint(t, spec, LintOptions{})

	after, err := os.ReadFile(spec)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != body {
		t.Errorf("file changed without --fix:\n%s", after)
	}
}
