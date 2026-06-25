package goss

import (
	"fmt"
	"log"

	"github.com/goss-org/goss/system"
	"github.com/goss-org/goss/util"
)

func runDiscoveryPhase(sys *system.System, discovery DiscoveryConfig, maxConcurrent int) (map[string]bool, error) {
	cfg := GossConfig{Discovery: discovery}
	return validateDiscovery(sys, cfg, maxConcurrent)
}

func loadGossConfigWithDiscover(c *util.Config) (*GossConfig, error) {
	if c.OutputFormat == "discovery" {
		return getGossConfig(c.VarsFiles, c.VarsInline, c.Spec, nil)
	}

	sys := system.New(c.PackageManager)

	if c.DiscoverSpec != "" {
		discoverCfg, err := getGossConfig(c.VarsFiles, c.VarsInline, c.DiscoverSpec, nil)
		if err != nil {
			return nil, fmt.Errorf("discover gossfile: %w", err)
		}
		if discoverCfg.Discovery.IsEmpty() {
			return nil, fmt.Errorf("discover gossfile %q has no discovery: tests", c.DiscoverSpec)
		}

		discovered, err := runDiscoveryPhase(sys, discoverCfg.Discovery, c.MaxConcurrent)
		if err != nil {
			return nil, fmt.Errorf("discover phase: %w", err)
		}

		if c.Spec != "" {
			peek, peekErr := getGossConfigPeek(c.VarsFiles, c.VarsInline, c.Spec)
			if peekErr == nil && !peek.Discovery.IsEmpty() {
				log.Printf("[INFO] ignoring inline discovery: in %q; using --discover %q", c.Spec, c.DiscoverSpec)
			}
		}

		return getGossConfig(c.VarsFiles, c.VarsInline, c.Spec, discovered)
	}

	peek, err := getGossConfigPeek(c.VarsFiles, c.VarsInline, c.Spec)
	if err != nil {
		return nil, err
	}

	if peek.Discovery.IsEmpty() {
		return getGossConfig(c.VarsFiles, c.VarsInline, c.Spec, nil)
	}

	discovered, err := runDiscoveryPhase(sys, peek.Discovery, c.MaxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("discover phase: %w", err)
	}

	return getGossConfig(c.VarsFiles, c.VarsInline, c.Spec, discovered)
}
