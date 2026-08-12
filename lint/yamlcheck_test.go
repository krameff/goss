package lint

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestCheckYAML(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		want     int
		wantLine int
		contains string
	}{
		{
			name: "valid",
			src:  "file:\n  /etc/hosts:\n    exists: true\n",
			want: 0,
		},
		{
			// A gossfile wrapped in a false {{ if }} renders to nothing. That
			// is normal, and goss runs it happily.
			name: "empty renders clean",
			src:  "",
			want: 0,
		},
		{
			name:     "duplicate key",
			src:      "port:\n  tcp:22:\n    listening: true\n    listening: false\n",
			want:     1,
			wantLine: 4,
			contains: `already defined at line 3`,
		},
		{
			name:     "syntax error",
			src:      "file:\n  x: [\n",
			want:     1,
			wantLine: 2,
			contains: "did not find expected node content",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckYAML("goss.yaml", "/tmp/rendered.yaml", []byte(tc.src))

			if len(got) != tc.want {
				t.Fatalf("got %d findings, want %d: %v", len(got), tc.want, got)
			}
			if tc.want == 0 {
				return
			}

			f := got[0]
			if f.Line != tc.wantLine {
				t.Errorf("line = %d, want %d", f.Line, tc.wantLine)
			}
			if !strings.Contains(f.Message, tc.contains) {
				t.Errorf("message = %q, want it to contain %q", f.Message, tc.contains)
			}
			if f.Space != SpaceRendered {
				t.Error("YAML findings are positions in the rendered output")
			}
			if f.Severity != SeverityError {
				t.Error("a file that doesn't parse is an error, not a warning")
			}
		})
	}
}

// go-yaml's TypeError carries several problems at once. Reporting only the
// first would mean one fix per run.
func TestFromYAMLErrorReportsEveryPosition(t *testing.T) {
	err := errors.New("yaml: unmarshal errors:\n  line 4: first problem\n  line 9: second problem")

	got := FromYAMLError("goss.yaml", "/tmp/r.yaml", RuleSchema, err)

	if len(got) != 2 {
		t.Fatalf("got %d findings, want 2: %v", len(got), got)
	}
	if got[0].Line != 4 || got[1].Line != 9 {
		t.Errorf("lines = %d, %d; want 4, 9", got[0].Line, got[1].Line)
	}
	if got[0].Rule != RuleSchema {
		t.Errorf("rule = %q, want %q", got[0].Rule, RuleSchema)
	}
}

// An error with no position still has to be reported, or a file that failed to
// load would produce nothing at all.
func TestFromYAMLErrorWithoutPosition(t *testing.T) {
	err := errors.New("invalid Attribute for Port:tcp:22: runing")

	got := FromYAMLError("goss.yaml", "/tmp/r.yaml", RuleSchema, err)

	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(got), got)
	}
	if got[0].Line != 1 {
		t.Errorf("line = %d, want 1", got[0].Line)
	}
	if !strings.Contains(got[0].Message, "runing") {
		t.Errorf("message = %q, want it to keep the detail", got[0].Message)
	}
}

func TestIsEmptyDocument(t *testing.T) {
	if !IsEmptyDocument(io.EOF) {
		t.Error("io.EOF means there was nothing to decode")
	}
	if IsEmptyDocument(errors.New("yaml: line 1: boom")) {
		t.Error("a real parse error is not an empty document")
	}
}

func TestAnchorKeys(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "aliased anchor block",
			src:  "defaults: &d\n  exists: true\nfile:\n  /x:\n    <<: *d\n",
			want: []string{"defaults"},
		},
		{
			// Nothing uses it, so it isn't a reusable block, it's just an
			// unknown key. Reporting it is correct.
			name: "anchor nobody aliases",
			src:  "defaults: &d\n  exists: true\nfile:\n  /x:\n    exists: true\n",
			want: nil,
		},
		{
			name: "no anchors at all",
			src:  "file:\n  /x:\n    exists: true\n",
			want: nil,
		},
		{
			// The anchor is buried, not on the top-level value itself.
			name: "nested anchor",
			src:  "defaults:\n  file: &d\n    exists: true\nfile:\n  /x:\n    <<: *d\n",
			want: []string{"defaults"},
		},
		{
			name: "not a mapping",
			src:  "- one\n- two\n",
			want: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AnchorKeys([]byte(tc.src))

			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for _, key := range tc.want {
				if !got[key] {
					t.Errorf("expected %q to be an anchor key, got %v", key, got)
				}
			}
		})
	}
}
