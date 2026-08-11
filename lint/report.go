package lint

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Output formats. text is for people, github annotates a pull request diff, and
// json is for anything else.
const (
	FormatText   = "text"
	FormatJSON   = "json"
	FormatGitHub = "github"
)

// Formats lists the supported values, for flag validation and help text.
func Formats() []string {
	return []string{FormatText, FormatJSON, FormatGitHub}
}

// Report writes findings in the requested format.
func Report(w io.Writer, findings []Finding, format string) error {
	Sort(findings)

	switch format {
	case FormatJSON:
		return reportJSON(w, findings)
	case FormatGitHub:
		return reportGitHub(w, findings)
	case FormatText, "":
		return reportText(w, findings)
	default:
		return fmt.Errorf("unknown format %q, want one of: %s", format, strings.Join(Formats(), ", "))
	}
}

func reportText(w io.Writer, findings []Finding) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(w, "no problems found")

		return err
	}

	// Findings on rendered output carry a line number for a file the user
	// didn't write, so say which file it belongs to rather than leaving them
	// to guess. One note per rendered file is enough.
	rendered := map[string]bool{}

	for _, f := range findings {
		where := ""
		if f.Space == SpaceRendered {
			where = " (rendered)"
			rendered[f.Rendered] = true
		}

		// Not every source of findings gives a column, and ":0" reads like a
		// real position rather than a missing one.
		position := fmt.Sprintf("%s:%d", f.File, f.Line)
		if f.Col > 0 {
			position = fmt.Sprintf("%s:%d", position, f.Col)
		}

		if _, err := fmt.Fprintf(w, "%s%s: %s: %s [%s]\n",
			position, where, f.Severity, f.Message, f.Rule); err != nil {
			return err
		}
	}

	// One note, however many files were checked. Listing a temp path per file
	// would drown the findings themselves.
	switch paths := sortedKeys(rendered); len(paths) {
	case 0:
	case 1:
		if _, err := fmt.Fprintf(w, "\nlines marked (rendered) refer to %s\n", paths[0]); err != nil {
			return err
		}
	default:
		if _, err := fmt.Fprintf(w, "\nlines marked (rendered) refer to each file's rendered output; "+
			"use --write-rendered DIR to keep them somewhere you can read\n"); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, "\n%s\n", summary(findings))

	return err
}

func reportJSON(w io.Writer, findings []Finding) error {
	// Marshal a non-nil slice so an empty result is [] rather than null.
	if findings == nil {
		findings = []Finding{}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	return enc.Encode(findings)
}

func reportGitHub(w io.Writer, findings []Finding) error {
	for _, f := range findings {
		level := "warning"
		if f.Severity == SeverityError {
			level = "error"
		}

		message := f.Message
		if f.Space == SpaceRendered {
			message = fmt.Sprintf("%s (line %d of the rendered output, %s)", message, f.Line, f.Rendered)
		}

		if _, err := fmt.Fprintf(w, "::%s file=%s,line=%d,col=%d::%s [%s]\n",
			level, f.File, f.Line, f.Col, message, f.Rule); err != nil {
			return err
		}
	}

	return nil
}

func summary(findings []Finding) string {
	var errs, warns int
	for _, f := range findings {
		if f.Severity == SeverityError {
			errs++
		} else {
			warns++
		}
	}

	return fmt.Sprintf("%s, %s", plural(errs, "error"), plural(warns, "warning"))
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}

	return fmt.Sprintf("%d %ss", n, word)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)

	return out
}
