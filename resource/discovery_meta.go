package resource

// DiscoveryMeta holds optional discovery and dependency metadata shared by resources.
type DiscoveryMeta struct {
	Register  string   `json:"register,omitempty" yaml:"register,omitempty"`
	DependsOn []string `json:"depends-on,omitempty" yaml:"depends-on,omitempty"`
}

func (d DiscoveryMeta) GetRegister() string {
	return d.Register
}

func (d DiscoveryMeta) GetDependsOn() []string {
	return d.DependsOn
}

// Discoverable resources expose a register name for discovery output.
type Discoverable interface {
	Resource
	GetRegister() string
}

// Dependent resources declare prerequisite tests that must pass first.
type Dependent interface {
	Resource
	GetDependsOn() []string
}
