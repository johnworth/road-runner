package dcompose

import (
	"fmt"
	"strconv"

	"github.com/cyverse-de/model"
	"github.com/spf13/viper"
)

// WORKDIR is the path to the working directory inside all of the containers
// that are run as part of a job.
const WORKDIR = "/de-app-work"

// CONFIGDIR is the path to the local configs inside the containers that are
// used to transfer files into and out of the job.
const CONFIGDIR = "/configs"

// VOLUMEDIR is the name of the directory that is used for the working directory
// volume.
const VOLUMEDIR = "workingvolume"

const (
	// TypeLabel is the label key applied to every container.
	TypeLabel = "org.iplantc.containertype"

	// InputContainer is the value used in the TypeLabel for input containers.
	InputContainer = iota

	// DataContainer is the value used in the TypeLabel for data containers.
	DataContainer

	// StepContainer is the value used in the TypeLabel for step containers.
	StepContainer

	// OutputContainer is the value used in the TypeLabel for output containers.
	OutputContainer
)

// Volume is a Docker volume definition in the Docker compose file.
type Volume struct {
	Driver  string
	Options map[string]string `yaml:"driver_opts"`
}

// Network is a Docker network definition in the docker-compose file.
type Network struct {
	Driver     string
	EnableIPv6 bool              `yaml:"enable_ipv6"`
	DriverOpts map[string]string `yaml:"driver_opts"`
}

// LoggingConfig configures the logging for a docker-compose service.
type LoggingConfig struct {
	Driver  string
	Options map[string]string `yaml:"driver_opts,omitempty"`
}

// ServiceNetworkConfig configures a docker-compose service to use a Docker
// Network.
type ServiceNetworkConfig struct {
	Aliases []string `yaml:",omitempty,flow"`
}

// Service configures a docker-compose service.
type Service struct {
	CapAdd        []string          `yaml:"cap_add,flow"`
	CapDrop       []string          `yaml:"cap_drop,flow"`
	Command       []string          `yaml:",omitempty,flow"`
	ContainerName string            `yaml:"container_name,omitempty"`
	DependsOn     []string          `yaml:"depends_on,omitempty,flow"`
	DNS           []string          `yaml:",omitempty,flow"`
	DNSSearch     []string          `yaml:"dns_search,omitempty,flow"`
	TMPFS         []string          `yaml:",omitempty,flow"`
	EntryPoint    string            `yaml:",omitempty"`
	Environment   map[string]string `yaml:",omitempty"`
	Expose        []string          `yaml:",omitempty,flow"`
	Image         string
	Labels        map[string]string                `yaml:",omitempty"`
	Logging       *LoggingConfig                   `yaml:",omitempty"`
	NetworkMode   string                           `yaml:"network_mode,omitempty"`
	Networks      map[string]*ServiceNetworkConfig `yaml:",omitempty"`
	Ports         []string                         `yaml:",omitempty,flow"`
	Volumes       []string                         `yaml:",omitempty,flow"`
	WorkingDir    string                           `yaml:"working_dir,omitempty"`
}

// JobCompose is the top-level type for what will become a job's docker-compose
// file.
type JobCompose struct {
	Version  string `yaml:"version"`
	Volumes  map[string]*Volume
	Networks map[string]*Network `yaml:",omitempty"`
	Services map[string]*Service
}

// New returns a newly instantiated *JobCompose instance.
func New() *JobCompose {
	return &JobCompose{
		Version:  "3.1",
		Volumes:  make(map[string]*Volume),
		Networks: make(map[string]*Network),
		Services: make(map[string]*Service),
	}
}

// InitFromJob fills out values as appropriate for running in the DE's Condor
// Cluster.
func (j *JobCompose) InitFromJob(job *model.Job, cfg *viper.Viper) {
	// Each job gets its own bridged network.
	j.Networks[job.InvocationID] = &Network{
		Driver: "bridge",
	}

	// The volume containing the local working directory
	j.Volumes[job.InvocationID] = &Volume{
		Driver: "local",
		Options: map[string]string{
			"type":   "none",
			"device": VOLUMEDIR,
			"o":      "bind",
		},
	}

	// Add the final output job
	j.Services[fmt.Sprintf("output-%s", job.InvocationID)] = &Service{
		CapAdd:  []string{"IPC_LOCK"},
		Image:   fmt.Sprintf("%s:%s", cfg.GetString("porklock.image"), cfg.GetString("porklock.tag")),
		Command: job.FinalOutputArguments(),
		Environment: map[string]string{
			"VAULT_ADDR":  cfg.GetString("vault.url"),
			"VAULT_TOKEN": cfg.GetString("vault.token"),
			"JOB_UUID":    job.InvocationID,
		},
		WorkingDir: WORKDIR,
		Volumes: []string{
			fmt.Sprintf("%s:%s:%s", job.InvocationID, WORKDIR, "rw"),
		},
		Networks: map[string]*ServiceNetworkConfig{
			job.InvocationID: &ServiceNetworkConfig{},
		},
		Labels: map[string]string{
			model.DockerLabelKey: job.InvocationID,
			TypeLabel:            strconv.Itoa(OutputContainer),
		},
		Logging: &LoggingConfig{Driver: "none"},
	}
}
