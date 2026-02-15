package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type OrganizationConfig struct {
	URL        string `json:"url"`
	BackupRoot string `json:"backupRoot"`
}

type Config struct {
	DefaultOrganization string                        `json:"defaultOrganization"`
	Organizations       map[string]OrganizationConfig `json:"organizations"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".azdo-vault", "config.json")
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	path := configPath()
	os.MkdirAll(filepath.Dir(path), 0700)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(path, data, 0600)
}

func (c *Config) ResolveOrganization(name string) (*OrganizationConfig, error) {
	if name != "" {
		org, ok := c.Organizations[name]
		if !ok {
			return nil, fmt.Errorf("organization '%s' not found", name)
		}
		return &org, nil
	}

	if c.DefaultOrganization == "" {
		return nil, fmt.Errorf("no default organization set")
	}

	org, ok := c.Organizations[c.DefaultOrganization]
	if !ok {
		return nil, fmt.Errorf("default organization '%s' not found", c.DefaultOrganization)
	}

	return &org, nil
}

func (c *Config) ResolveOrganizationWithName(name string) (string, *OrganizationConfig, error) {

	if name != "" {
		org, ok := c.Organizations[name]
		if !ok {
			return "", nil, fmt.Errorf("organization '%s' not found", name)
		}
		return name, &org, nil
	}

	if c.DefaultOrganization == "" {
		return "", nil, fmt.Errorf("no default organization set")
	}

	org, ok := c.Organizations[c.DefaultOrganization]
	if !ok {
		return "", nil, fmt.Errorf("default organization '%s' not found", c.DefaultOrganization)
	}

	return c.DefaultOrganization, &org, nil
}
