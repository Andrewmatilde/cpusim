package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Host struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
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
	return fmt.Sprintf("http://%s:80", h.IP)
}

func (h *Host) GetCollectorServiceURL() string {
	return fmt.Sprintf("http://%s:8080", h.IP)
}