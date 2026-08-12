package lint

import "testing"

func TestRenameFor(t *testing.T) {
	tests := []struct {
		notice string
		want   string
	}{
		{"please use `toUpper` instead", "toUpper"},
		{"use `sha1Sum` instead.", "sha1Sum"},
		{"please use `base64Encode` instead", "base64Encode"},
		// Not a rename: the call shape changes, so swapping the name would
		// leave a file that renders differently.
		{"use new native syntax `{{ $dict | dig \"key\" }}` instead of old Sprig's", ""},
		{"", ""},
		{"please use `toUpper` or `upperCase` instead", ""},
	}

	for _, tc := range tests {
		if got := renameFor(tc.notice); got != tc.want {
			t.Errorf("renameFor(%q) = %q, want %q", tc.notice, got, tc.want)
		}
	}
}

func TestFix(t *testing.T) {
	src := []byte(`a: {{ "x" | upper }}
b: {{ b64enc "y" }}
`)
	findings := CheckTemplate("g.yaml", src, testFuncs())

	out, applied := Fix(src, findings)

	if len(applied) != 2 {
		t.Fatalf("applied %d fixes, want 2", len(applied))
	}

	want := `a: {{ "x" | toUpper }}
b: {{ base64Encode "y" }}
`
	if string(out) != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

// Fixing several names on one line must not corrupt the later ones: the edits
// are applied back to front so earlier offsets stay valid.
func TestFixMultiplePerLine(t *testing.T) {
	src := []byte(`a: {{ upper (b64enc (toYaml .X)) }}` + "\n")
	findings := CheckTemplate("g.yaml", src, testFuncs())

	out, applied := Fix(src, findings)

	if len(applied) != 3 {
		t.Fatalf("applied %d fixes, want 3: %s", len(applied), out)
	}

	want := `a: {{ toUpper (base64Encode (toYAML .X)) }}` + "\n"
	if string(out) != want {
		t.Errorf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestFixLeavesSignatureChangesAlone(t *testing.T) {
	src := []byte(`a: {{ dig "k" "d" .X }}` + "\n")
	findings := CheckTemplate("g.yaml", src, testFuncs())

	out, applied := Fix(src, findings)

	if len(applied) != 0 {
		t.Errorf("dig changed signature, it must not be renamed automatically")
	}
	if string(out) != string(src) {
		t.Errorf("source was modified:\n%s", out)
	}
}

func TestFixIgnoresNonDeprecatedFindings(t *testing.T) {
	findings := []Finding{
		{Rule: RuleYAML, Message: "trailing spaces", Line: 4},
		{Rule: RuleTemplateParse, Message: "function not defined", Line: 1},
	}

	src := []byte("a: 1\n")
	out, applied := Fix(src, findings)

	if len(applied) != 0 || string(out) != string(src) {
		t.Error("only deprecated-func findings should be fixable")
	}
}
