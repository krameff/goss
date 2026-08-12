package lint

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// yamlErrLineRe matches the position in a go-yaml error. They come in two
// shapes, a single error:
//
//	yaml: line 4: did not find expected node content
//
// and a TypeError, which is a header followed by one indented line each:
//
//	yaml: unmarshal errors:
//	  line 4: mapping key "listening" already defined at line 3
var yamlErrLineRe = regexp.MustCompile(`^(?:yaml: )?line (\d+): (.*)$`)

// CheckYAML parses the rendered output and reports anything that stops it being
// read as YAML at all: syntax errors, and duplicate keys, which go-yaml treats
// as an error rather than silently keeping the last one.
//
// This is the structural half of the YAML checks, and it is deliberately pure
// Go so it always runs. Yamllint covers style on top of it, but it is an
// external Python tool that most machines don't have, and "is this file even
// parseable" is too important to depend on that.
//
// Line numbers are in rendered space, like every other YAML finding.
func CheckYAML(sourceFile, renderedPath string, rendered []byte) []Finding {
	var v any

	dec := yaml.NewDecoder(bytes.NewReader(rendered))
	if err := dec.Decode(&v); err != nil && !IsEmptyDocument(err) {
		return FromYAMLError(sourceFile, renderedPath, RuleYAML, err)
	}

	return nil
}

// IsEmptyDocument reports whether a decode failed only because there was
// nothing to decode. go-yaml signals that with io.EOF.
//
// A gossfile that renders to nothing is normal, not a mistake: a file wrapped
// in a single {{ if }} does exactly that whenever the condition is false, and
// goss itself is happy to run it.
func IsEmptyDocument(err error) bool {
	return errors.Is(err, io.EOF)
}

// AnchorKeys returns the top-level keys that exist only to be reused, which is
// to say the ones defining a YAML anchor that something else in the document
// aliases:
//
//	defaults: &defaults        <- this key
//	  exists: true
//
//	file:
//	  /etc/passwd:
//	    <<: *defaults
//
// YAML has no other way to write a reusable block, so the block has to sit
// somewhere, and at the top of the file is where people put it. goss ignores
// top-level keys it doesn't recognise, so these have always worked.
//
// A checker that doesn't know this reports the block as a misspelled resource
// type, which is both wrong and unfixable: there is nothing to correct.
//
// Requiring the anchor to actually be aliased is what keeps this from being a
// hole. `fille:` defines no anchor, so it is still reported.
func AnchorKeys(rendered []byte) map[string]bool {
	var doc yaml.Node
	if err := yaml.Unmarshal(rendered, &doc); err != nil || len(doc.Content) == 0 {
		return nil
	}

	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}

	aliased := map[string]bool{}
	walkNode(&doc, func(n *yaml.Node) {
		if n.Kind == yaml.AliasNode {
			aliased[n.Value] = true
		}
	})

	out := map[string]bool{}
	// A mapping's Content is a flat key, value, key, value list.
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i], root.Content[i+1]
		walkNode(value, func(n *yaml.Node) {
			if n.Anchor != "" && aliased[n.Anchor] {
				out[key.Value] = true
			}
		})
	}

	return out
}

func walkNode(n *yaml.Node, fn func(*yaml.Node)) {
	if n == nil {
		return
	}
	fn(n)
	for _, c := range n.Content {
		walkNode(c, fn)
	}
}

// FromYAMLError turns a go-yaml error into findings, one per position it
// reports. A TypeError carries several at once, and reporting only the first
// would mean the user fixes one problem per run.
//
// An error with no position at all still produces a finding, at line 1, since
// dropping it would mean reporting nothing for a file that failed to load.
func FromYAMLError(sourceFile, renderedPath, rule string, err error) []Finding {
	var findings []Finding

	for _, line := range strings.Split(err.Error(), "\n") {
		m := yamlErrLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}

		lineNo, _ := strconv.Atoi(m[1])
		findings = append(findings, Finding{
			File:     sourceFile,
			Line:     lineNo,
			Rule:     rule,
			Message:  m[2],
			Severity: SeverityError,
			Space:    SpaceRendered,
			Rendered: renderedPath,
		})
	}

	if len(findings) == 0 {
		findings = append(findings, Finding{
			File:     sourceFile,
			Line:     1,
			Rule:     rule,
			Message:  cleanYAMLError(err.Error()),
			Severity: SeverityError,
			Space:    SpaceRendered,
			Rendered: renderedPath,
		})
	}

	return findings
}

// cleanYAMLError strips go-yaml's framing from a message that had no position,
// so the user reads the problem rather than the library's plumbing.
func cleanYAMLError(msg string) string {
	msg = strings.TrimPrefix(msg, "yaml: unmarshal errors:")
	msg = strings.TrimPrefix(msg, "yaml: ")

	return strings.TrimSpace(strings.ReplaceAll(msg, "\n", "; "))
}
