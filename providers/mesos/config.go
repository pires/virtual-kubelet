package mesos

import (
	"fmt"
	"github.com/virtual-kubelet/virtual-kubelet/providers/mesos/scheduler"
	"io"

	"github.com/BurntSushi/toml"

	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// Provider configuration defaults.
	defaultCPUCapacity     = "20"
	defaultMemoryCapacity  = "100Gi"
	defaultStorageCapacity = "40Gi"
	defaultPodCapacity     = "20"

	// Provider configuration minimum.
	minCPUCapacity    = "250m"
	minMemoryCapacity = "512Mi"
	minPodCapacity    = "1"
)

// providerConfig contains virtual-kubelet Mesos provider configurable parameters.
type providerConfig struct {
	CPU       string            `json:"cpu,omitempty"`
	Memory    string            `json:"memory,omitempty"`
	Storage   string            `json:"storage,omitempty"`
	Pods      string            `json:"pods,omitempty"`
	Scheduler *scheduler.Config `json:"scheduler,omitempty"`
}

// loadConfig tries and decode the provider configuration from a toml file.
func (p *Provider) loadConfig(r io.Reader) error {
	var config providerConfig
	var q resource.Quantity
	var err error

	// Set defaults for optional fields.
	config.CPU = defaultCPUCapacity
	config.Memory = defaultMemoryCapacity
	config.Storage = defaultStorageCapacity
	config.Pods = defaultPodCapacity
	config.Scheduler = scheduler.DefaultConfig()

	// Load configuration file.
	if _, err := toml.DecodeReader(r, &config); err != nil {
		return err
	}

	// Validate advertised capacity.
	if q, err = resource.ParseQuantity(config.CPU); err != nil {
		return fmt.Errorf("invalid CPU value %v", config.CPU)
	}
	if q.Cmp(resource.MustParse(minCPUCapacity)) == -1 {
		return fmt.Errorf("CPU value %v is less than the minimum %v", config.CPU, minCPUCapacity)
	}
	if q, err = resource.ParseQuantity(config.Memory); err != nil {
		return fmt.Errorf("Invalid memory value %v", config.Memory)
	}
	if q.Cmp(resource.MustParse(minMemoryCapacity)) == -1 {
		return fmt.Errorf("Memory value %v is less than the minimum %v", config.Memory, minMemoryCapacity)
	}
	if q, err = resource.ParseQuantity(config.Storage); err != nil {
		return fmt.Errorf("Invalid storage value %v", config.Storage)
	}
	if q, err = resource.ParseQuantity(config.Pods); err != nil {
		return fmt.Errorf("Invalid pods value %v", config.Pods)
	}
	if q.Cmp(resource.MustParse(minPodCapacity)) == -1 {
		return fmt.Errorf("Pod value %v is less than the minimum %v", config.Pods, minPodCapacity)
	}

	p.config = &config

	return nil
}
