package goss

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goss-org/goss/resource"
	"github.com/goss-org/goss/system"
	"github.com/goss-org/goss/util"
)

func TestValidateDiscoveryFormat(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "discovery.yaml")
	content := []byte(`discovery:
  file:
    /etc/hosts:
      register: hosts_exists
      exists: true
`)
	if err := os.WriteFile(spec, content, 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cfg, err := util.NewConfig(
		util.WithSpecFile(spec),
		util.WithOutputFormat("discovery"),
		util.WithNoColor(),
	)
	if err != nil {
		t.Fatalf("new config: %v", err)
	}

	code, err := Validate(cfg)
	if err != nil {
		t.Fatalf("validate discovery: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestValidateDependsOnSkipsDependent(t *testing.T) {
	sys := system.New("")
	base := &resource.File{
		Exists: true,
		Path:   "/nonexistent/goss-dependency-base",
	}
	dependent := &resource.File{
		Exists: true,
		DiscoveryMeta: resource.DiscoveryMeta{
			DependsOn: []string{"base"},
		},
	}
	base.SetID("base")
	dependent.SetID("dependent")
	resources := []resource.Resource{base, dependent}

	out, err := validateWithDependencies(sys, resources, 1)
	if err != nil {
		t.Fatalf("validate with dependencies: %v", err)
	}

	var skipped bool
	var failed bool
	for group := range out {
		for _, result := range group {
			if result.Skipped && result.Property == "depends-on" {
				skipped = true
			}
			if result.ResourceType == "File" && result.Property == "exists" && result.Result == resource.FAIL {
				failed = true
			}
		}
	}

	if !failed {
		t.Fatal("expected base resource to fail")
	}
	if !skipped {
		t.Fatal("expected dependent resource to be skipped")
	}
}

func TestTemplateDiscoveredVars(t *testing.T) {
	filter, err := NewTemplateFilter([]string{}, `{"Discovered":{"enabled":true}}`, nil)
	if err != nil {
		t.Fatalf("template filter: %v", err)
	}

	out, err := filter([]byte(`enabled: {{ .Discovered.enabled }}`))
	if err != nil {
		t.Fatalf("render template: %v", err)
	}

	if string(out) != "enabled: true" {
		t.Fatalf("unexpected rendered output: %q", string(out))
	}
}

func TestValidateWithoutDiscover(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "goss.yml")
	if err := os.WriteFile(spec, []byte(`file:
  /etc/hosts:
    exists: true
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cfg, err := util.NewConfig(
		util.WithSpecFile(spec),
		util.WithOutputFormat("documentation"),
		util.WithNoColor(),
	)
	if err != nil {
		t.Fatalf("new config: %v", err)
	}
	if cfg.DiscoverSpec != "" {
		t.Fatal("expected empty DiscoverSpec for plain validate")
	}

	code, err := Validate(cfg)
	if err != nil {
		t.Fatalf("validate without discover: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestValidateWithoutDiscoverErrorsOnMissingTemplateVar(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "goss.yml")
	if err := os.WriteFile(spec, []byte(`file:
  {{ .Vars.missing_key }}/hosts:
    exists: true
`), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	cfg, err := util.NewConfig(util.WithSpecFile(spec), util.WithNoColor())
	if err != nil {
		t.Fatalf("new config: %v", err)
	}

	_, err = Validate(cfg)
	if err == nil {
		t.Fatal("expected template error for missing .Vars key without discover")
	}
}

func discoveryExamplesDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join("integration-tests", "goss", "examples", "discovery")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("discovery examples missing: %v", err)
	}
	return dir
}

func TestValidateWithDiscoverFlag(t *testing.T) {
	dir := discoveryExamplesDir(t)
	cfg, err := util.NewConfig(
		util.WithSpecFile(filepath.Join(dir, "goss.yml")),
		util.WithDiscoverFile(filepath.Join(dir, "discovery.yaml")),
		util.WithOutputFormat("documentation"),
		util.WithNoColor(),
	)
	if err != nil {
		t.Fatalf("new config: %v", err)
	}

	code, err := Validate(cfg)
	if err != nil {
		t.Fatalf("validate with discover: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestValidateInlineDiscovery(t *testing.T) {
	dir := discoveryExamplesDir(t)
	cfg, err := util.NewConfig(
		util.WithSpecFile(filepath.Join(dir, "goss-inline.yml")),
		util.WithOutputFormat("documentation"),
		util.WithNoColor(),
	)
	if err != nil {
		t.Fatalf("new config: %v", err)
	}

	code, err := Validate(cfg)
	if err != nil {
		t.Fatalf("validate inline discovery: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestDiscoverFlagOverridesInline(t *testing.T) {
	dir := t.TempDir()
	inline := filepath.Join(dir, "inline.yml")
	discover := filepath.Join(dir, "discover.yaml")
	main := filepath.Join(dir, "main.yml")

	inlineContent := []byte(`discovery:
  file:
    /etc/hosts:
      register: from_inline
      exists: true
file:
  /tmp/inline-should-not-run:
    exists: false
`)
	discoverContent := []byte(`discovery:
  file:
    /etc/hosts:
      register: from_flag
      exists: true
`)
	mainContent := []byte(`{{ if .Discovered.from_flag }}
file:
  /etc/hosts:
    exists: true
{{ end }}
`)

	for path, content := range map[string][]byte{
		inline: inlineContent, discover: discoverContent, main: mainContent,
	} {
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	cfg, err := util.NewConfig(
		util.WithSpecFile(main),
		util.WithDiscoverFile(discover),
		util.WithOutputFormat("documentation"),
		util.WithNoColor(),
	)
	if err != nil {
		t.Fatalf("new config: %v", err)
	}

	// Peek: inline file has discovery but we use --discover pointing elsewhere
	peekCfg, err := util.NewConfig(util.WithSpecFile(inline), util.WithNoColor())
	if err != nil {
		t.Fatalf("peek config: %v", err)
	}
	peek, err := getGossConfigPeek(peekCfg.VarsFiles, peekCfg.VarsInline, peekCfg.Spec)
	if err != nil {
		t.Fatalf("peek load: %v", err)
	}
	if peek.Discovery.IsEmpty() {
		t.Fatal("expected inline discovery in peek file")
	}

	code, err := Validate(cfg)
	if err != nil {
		t.Fatalf("validate override: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0 when --discover supplies from_flag, got %d", code)
	}
}

func TestValidateDiscoverWithDependsOn(t *testing.T) {
	dir := discoveryExamplesDir(t)
	cfg, err := util.NewConfig(
		util.WithSpecFile(filepath.Join(dir, "goss-with-deps.yml")),
		util.WithDiscoverFile(filepath.Join(dir, "discovery.yaml")),
		util.WithOutputFormat("documentation"),
		util.WithNoColor(),
	)
	if err != nil {
		t.Fatalf("new config: %v", err)
	}

	code, err := Validate(cfg)
	if err != nil {
		t.Fatalf("validate discover with depends-on: %v", err)
	}
	if code == 0 {
		t.Fatal("expected non-zero exit when prerequisite fails")
	}
}
