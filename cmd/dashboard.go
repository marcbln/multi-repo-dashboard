package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/multi-repo-dashboard/internal/config"
	"github.com/yourusername/multi-repo-dashboard/internal/tui"
)

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Short:   "Launch the TUI dashboard",
	Aliases: []string{"dash", "ui"},
	RunE: func(cmd *cobra.Command, args []string) error {
		var cfg config.Config
		if err := viper.Unmarshal(&cfg); err != nil {
			return fmt.Errorf("failed to parse config: %w", err)
		}

		if len(cfg.Repositories) == 0 {
			// Check if config file exists
			configFile := viper.ConfigFileUsed()
			if configFile == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("no repositories configured and couldn't determine home directory")
				}
				configFile = filepath.Join(home, ".config", "mrd", "config.yaml")
			}

			// Create default config if it doesn't exist
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				if err := config.CreateDefaultConfig(configFile); err != nil {
					return fmt.Errorf("no repositories configured. Please add them to your config file at %s", configFile)
				}
				return fmt.Errorf("no repositories configured. Created default config at %s. Please edit it and add your repositories, or use 'mrd repo add' command", configFile)
			}

			return fmt.Errorf("no repositories configured. Please add them to your config file at %s, or use 'mrd repo add' command", configFile)
		}

		p := tea.NewProgram(tui.NewModel(&cfg), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running dashboard: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}
