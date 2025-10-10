package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Host type constants
const (
	HostTypeTarget = "target"
	HostTypeClient = "client"
)

type Host struct {
	Name       string `json:"name"`
	ExternalIP string `json:"externalIP"`
	InternalIP string `json:"internalIP,omitempty"`
	HostType   string `json:"hostType"` // "target" or "client"
}

type Config struct {
	Hosts []Host `json:"hosts"`
}

func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func (c *Config) GetHostByName(name string) *Host {
	for _, host := range c.Hosts {
		if host.Name == name {
			return &host
		}
	}
	return nil
}

func (c *Config) GetAllHosts() []Host {
	return c.Hosts
}

func (h *Host) GetCPUServiceURL() string {
	// cpusim-server runs on port 80 on target hosts
	return fmt.Sprintf("http://%s:80", h.ExternalIP)
}

func (h *Host) GetCollectorServiceURL() string {
	// collector-server runs on port 8080 on target hosts
	return fmt.Sprintf("http://%s:8080", h.ExternalIP)
}

func (h *Host) GetRequesterServiceURL() string {
	// requester-server runs on port 80 on client hosts
	return fmt.Sprintf("http://%s:80", h.ExternalIP)
}

func (h *Host) IsTarget() bool {
	return h.HostType == HostTypeTarget
}

func (h *Host) IsClient() bool {
	return h.HostType == HostTypeClient
}
