package config

// Config represents the root YAML configuration
type Config struct {
	Repositories []RepoConfig `mapstructure:"repositories"`
}

// RepoConfig represents a single tracked git repository
type RepoConfig struct {
	Name string `mapstructure:"name"`
	Path string `mapstructure:"path"`
}
