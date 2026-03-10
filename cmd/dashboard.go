package cmd

import (
	"fmt"

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
			return fmt.Errorf("no repositories configured. Please add them to your config file")
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
