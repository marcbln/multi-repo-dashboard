package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/multi-repo-dashboard/internal/config"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repository configuration",
	Long:  `Add, list, or remove repositories from the configuration.`,
}

var repoAddCmd = &cobra.Command{
	Use:   "add [name] [path]",
	Short: "Add a repository to the configuration",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		path := args[1]

		// Convert path to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		// Check if path exists and is a directory
		if info, err := os.Stat(absPath); err != nil {
			return fmt.Errorf("path %s does not exist: %w", absPath, err)
		} else if !info.IsDir() {
			return fmt.Errorf("path %s is not a directory", absPath)
		}

		// Get config file path
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("couldn't determine home directory: %w", err)
			}
			configFile = filepath.Join(home, ".config", "mrd", "config.yaml")
		}

		// Create config if it doesn't exist
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			if err := config.CreateDefaultConfig(configFile); err != nil {
				return fmt.Errorf("failed to create default config: %w", err)
			}
			fmt.Printf("Created default config at %s\n", configFile)
		}

		// Load existing config
		cfg, err := config.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Add repository
		if err := cfg.AddRepository(name, absPath); err != nil {
			return fmt.Errorf("failed to add repository: %w", err)
		}

		// Save config
		if err := config.SaveConfig(configFile, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Added repository '%s' at path '%s'\n", name, absPath)
		return nil
	},
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg config.Config
		if err := viper.Unmarshal(&cfg); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		if len(cfg.Repositories) == 0 {
			fmt.Println("No repositories configured.")
			return nil
		}

		fmt.Println("Configured repositories:")
		for i, repo := range cfg.Repositories {
			fmt.Printf("  %d. %s: %s\n", i+1, repo.Name, repo.Path)
		}

		return nil
	},
}

func init() {
	repoCmd.AddCommand(repoAddCmd)
	repoCmd.AddCommand(repoListCmd)
	rootCmd.AddCommand(repoCmd)
}
