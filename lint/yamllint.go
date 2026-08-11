package lint

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ErrYamllintMissing is returned when yamllint isn't installed and the caller
// asked for it to be required.
var ErrYamllintMissing = errors.New("yamllint is not installed or not on PATH")

// yamllintLineRe matches yamllint's --format parsable output:
//
//	file.yaml:12:3: [warning] too many spaces (rule-name)
var yamllintLineRe = regexp.MustCompile(`^(.*?):(\d+):(\d+): \[(\w+)\] (.*)$`)

// defaultYamllintConfig is used when the user hasn't got a yamllint config of
// their own. It is yamllint's defaults with two rules turned off that fit
// yamllint's assumptions rather than goss's:
//
//   - document-start: gossfiles don't begin with "---", so leaving this on
//     means a warning on essentially every file. goss's own .yamllint disables
//     it for the same reason.
//   - line-length: gossfile values are frequently long commands or paths, and
//     wrapping them isn't an option.
//
// Anyone who wants the stock rules can point --yamllint-config at their own
// config, which takes over completely.
const defaultYamllintConfig = `{extends: default, rules: {document-start: disable, line-length: disable}}`

// YamllintOptions controls the YAML half of a lint run.
type YamllintOptions struct {
	// Config is an explicit yamllint config file. Empty means let yamllint
	// find its own, after ConfigFor has looked for one near the gossfile.
	Config string
	// Required makes a missing yamllint an error rather than a skip. CI
	// should set this, so the checks can't silently stop running.
	Required bool
}

// YamllintAvailable reports whether yamllint can be run. Callers use it to tell
// the user the YAML checks were skipped, rather than letting a clean report
// imply the file was fully checked.
func YamllintAvailable() bool {
	_, err := exec.LookPath("yamllint")

	return err == nil
}

// Yamllint runs yamllint over already-rendered YAML and turns its output into
// findings.
//
// yamllint is a Python tool and goss is a static binary, so this shells out
// rather than reimplementing it. If yamllint isn't installed the YAML checks are
// skipped and the template checks still run, unless opts.Required is set.
//
// renderedPath is the file yamllint reads. sourceFile is what the findings are
// labelled with, since that's the file the user knows about. The line numbers
// stay in rendered space, which is why every finding here is SpaceRendered.
func Yamllint(sourceFile, renderedPath string, opts YamllintOptions) ([]Finding, error) {
	if !YamllintAvailable() {
		if opts.Required {
			return nil, ErrYamllintMissing
		}

		return nil, nil
	}

	args := []string{"--format", "parsable"}
	if opts.Config != "" {
		args = append(args, "--config-file", opts.Config)
	} else {
		args = append(args, "--config-data", defaultYamllintConfig)
	}
	args = append(args, renderedPath)

	// yamllint exits non-zero when it finds problems, which is not a failure
	// of the run. Only a missing/!ExitError failure is.
	out, err := exec.Command("yamllint", args...).Output()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return nil, err
	}

	return parseYamllint(sourceFile, renderedPath, string(out)), nil
}

func parseYamllint(sourceFile, renderedPath, out string) []Finding {
	var findings []Finding

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		m := yamllintLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}

		severity := SeverityWarning
		if m[4] == "error" {
			severity = SeverityError
		}

		lineNo, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])

		findings = append(findings, Finding{
			File:     sourceFile,
			Line:     lineNo,
			Col:      col,
			Rule:     RuleYAML,
			Message:  m[5],
			Severity: severity,
			Space:    SpaceRendered,
			Rendered: renderedPath,
		})
	}

	return findings
}

// ConfigFor looks for a .yamllint config near the gossfile, walking up to the
// filesystem root. Users already have their YAML style encoded in one, and it
// would be rude to ignore it.
//
// Returns "" when there's nothing to find, which leaves yamllint to its own
// discovery rules.
func ConfigFor(gossfile string) string {
	dir, err := filepath.Abs(filepath.Dir(gossfile))
	if err != nil {
		return ""
	}

	for {
		for _, name := range []string{".yamllint", ".yamllint.yaml", ".yamllint.yml"} {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
