package dcompose

// JobCompose is the top-level type for what will become a job's docker-compose
// file.
type JobCompose struct {
	Version string `yaml:"version"`

	Volumes map[string]struct {
		Driver     string
		DriverOpts map[string]string `yaml:"driver_opts"`
	}

	Networks map[string]struct {
		Driver     string
		EnableIPv6 bool              `yaml:"enable_ipv6"`
		DriverOpts map[string]string `yaml:"driver_opts"`
	} `yaml:",omitempty"`

	Services map[string]struct {
		CapAdd        []string          `yaml:"cap_add,flow"`
		CapDrop       []string          `yaml:"cap_drop,flow"`
		Command       string            `yaml:",omitempty"`
		ContainerName string            `yaml:"container_name,omitempty"`
		DependsOn     []string          `yaml:"depends_on,omitempty,flow"`
		DNS           []string          `yaml:",omitempty,flow"`
		DNSSearch     []string          `yaml:"dns_search,omitempty,flow"`
		TMPFS         []string          `yaml:",omitempty,flow"`
		EntryPoint    string            `yaml:",omitempty"`
		Environment   map[string]string `yaml:",omitempty"`
		Expose        []string          `yaml:",omitempty,flow"`
		Image         string
		Labels        map[string]string `yaml:",omitempty"`

		Logging struct {
			Driver  string
			Options map[string]string `yaml:"driver_opts,omitempty"`
		} `yaml:",omitempty"`

		NetworkMode string `yaml:"network_mode,omitempty"`

		Networks map[string]struct {
			Aliases []string `yaml:",omitempty,flow"`
		}

		Ports      []string `yaml:",omitempty,flow"`
		Volumes    []string `yaml:",omitempty,flow"`
		WorkingDir string   `yaml:"working_dir,omitempty"`
	}
}
