package lint

import (
	"regexp"
	"sort"
)

// renameRe matches a notice that is purely a rename, e.g.
//
//	please use `toUpper` instead
//	use `sha1Sum` instead.
//
// Anything else is left alone. Some notices describe a changed signature
// rather than a new name -- `dig` moved from `{{ dig "key" "default" $dict }}`
// to `{{ $dict | dig "key" }}` -- and swapping the identifier there would
// produce a file that parses and renders differently. A fixer that only
// handles the unambiguous cases is worth having; one that guesses is not.
var renameRe = regexp.MustCompile("^(?:please )?use `([A-Za-z0-9_]+)` instead\\.?$")

// renameFor returns the new name for a deprecated function, or "" when the
// notice isn't a straight rename.
func renameFor(notice string) string {
	m := renameRe.FindStringSubmatch(notice)
	if m == nil {
		return ""
	}

	return m[1]
}

// Fixable reports whether --fix can do anything with this finding.
func Fixable(f Finding) bool {
	return f.Rule == RuleDeprecatedFunc && f.Replacement != ""
}

// Fix rewrites src, replacing each fixable finding's function name with its
// replacement, and returns the new content along with the findings it applied.
//
// Only findings carrying a byte offset and a replacement are touched, so
// anything the checks couldn't place exactly is left alone. Edits are applied
// back to front so earlier offsets stay valid.
//
// The caller is expected to verify the result before keeping it: see
// docs/lint.md, and Lint's --fix path, which re-renders and refuses to write
// if the output changed.
func Fix(src []byte, findings []Finding) ([]byte, []Finding) {
	var todo []Finding
	for _, f := range findings {
		if Fixable(f) {
			todo = append(todo, f)
		}
	}
	if len(todo) == 0 {
		return src, nil
	}

	// Back to front, so applying one edit doesn't shift the next one's offset.
	sort.Slice(todo, func(i, j int) bool { return todo[i].Offset > todo[j].Offset })

	out := src
	var applied []Finding
	for _, f := range todo {
		end := f.Offset + len(f.Symbol)
		if f.Offset < 0 || end > len(out) || string(out[f.Offset:end]) != f.Symbol {
			// The offset doesn't point at the name we expected, so don't
			// touch it. Better to report the finding than corrupt the file.
			continue
		}

		buf := make([]byte, 0, len(out)-len(f.Symbol)+len(f.Replacement))
		buf = append(buf, out[:f.Offset]...)
		buf = append(buf, f.Replacement...)
		buf = append(buf, out[end:]...)
		out = buf
		applied = append(applied, f)
	}

	return out, applied
}
