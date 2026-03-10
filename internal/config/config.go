package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the root YAML configuration
type Config struct {
	Repositories []RepoConfig `mapstructure:"repositories"`
}

// RepoConfig represents a single tracked git repository
type RepoConfig struct {
	Name string `mapstructure:"name"`
	Path string `mapstructure:"path"`
}

// CreateDefaultConfig creates a default config file at the specified path
func CreateDefaultConfig(configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config with example repositories
	defaultConfig := Config{
		Repositories: []RepoConfig{
			{
				Name: "example-repo",
				Path: "/path/to/your/repository",
			},
		},
	}

	// Marshal to YAML
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadConfig loads config from the specified path
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// SaveConfig saves config to the specified path
func SaveConfig(configPath string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddRepository adds a new repository to the config
func (cfg *Config) AddRepository(name, path string) error {
	// Check for duplicate names
	for _, repo := range cfg.Repositories {
		if repo.Name == name {
			return fmt.Errorf("repository with name '%s' already exists", name)
		}
	}

	cfg.Repositories = append(cfg.Repositories, RepoConfig{
		Name: name,
		Path: path,
	})

	return nil
}
