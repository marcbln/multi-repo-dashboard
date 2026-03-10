---
filename: "_ai/backlog/active/260310_1913__IMPLEMENTATION_PLAN__git-agent-tui.md"
title: "Multi-Repo AI Git Dashboard (TUI)"
createdAt: 2026-03-10 19:13
updatedAt: 2026-03-10 19:13
status: draft
priority: high
tags: [golang, cli, tui, bubbletea, git, automation]
estimatedComplexity: complex
documentType: IMPLEMENTATION_PLAN
---

## 1. Problem Description
When managing multiple local Git repositories—especially when autonomous AI agents are modifying files in the background—it is easy to lose track of uncommitted changes (`dirty` states) and remote sync status. The user needs a centralized, tiling Terminal User Interface (TUI) dashboard to monitor a curated list of local Git repositories (defined via a YAML config). The dashboard must visually indicate dirty states, unpushed/unpulled commits, and provide actionable shortcuts (e.g., "Pull All") to keep local repositories in sync with their remotes. This tool will serve as the foundational UI for a larger multi-repo AI-coding orchestrator.

## 2. Project Environment Details
```
- Project Name: multi-repo-ai-orchestrator (Executable: mrao)
- Language: Go 1.23+
- Framework: Cobra (CLI), Viper (Config)
- TUI Stack: Bubble Tea (Core), Bubbles (Components), Lipgloss (Styling)
- Config Format: YAML
- Key Dependencies:
  - github.com/spf13/cobra v1.8.1
  - github.com/spf13/viper v1.19.0
  - github.com/charmbracelet/bubbletea v1.1.0
  - github.com/charmbracelet/bubbles v0.20.0
  - github.com/charmbracelet/lipgloss/v2 v2.0.1
```

---

## Phase 1: Project Setup & CLI Foundation

This phase establishes the Go standard layout and initializes the Cobra/Viper CLI foundation.

```go
// [NEW FILE] go.mod
module github.com/yourusername/multi-repo-ai-orchestrator

go 1.23

require (
	github.com/charmbracelet/bubbles v0.20.0
	github.com/charmbracelet/bubbletea v1.1.0
	github.com/charmbracelet/lipgloss/v2 v2.0.1
	github.com/spf13/cobra v1.8.1
	github.com/spf13/viper v1.19.0
)
```

```go
// [NEW FILE] main.go
package main

import (
	"fmt"
	"os"

	"github.com/yourusername/multi-repo-ai-orchestrator/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

```go
// [NEW FILE] internal/config/config.go
package config

// Config represents the root YAML configuration
type Config struct {
	Repositories[]RepoConfig `mapstructure:"repositories"`
}

// RepoConfig represents a single tracked git repository
type RepoConfig struct {
	Name string `mapstructure:"name"`
	Path string `mapstructure:"path"`
}
```

```go
// [NEW FILE] cmd/root.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "mrao",
	Short: "Multi-Repo AI Orchestrator Dashboard",
	Long:  `A TUI dashboard for monitoring and orchestrating multi-repository AI coding agents.`,
	PersistentPreRunE: func(cmd *cobra.Command, args[]string) error {
		return initConfig()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/mrao/config.yaml)")
}

func initConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		viper.AddConfigPath(filepath.Join(home, ".config", "mrao"))
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
		// It's okay if config doesn't exist yet, we can create an empty one later
	}
	return nil
}
```

## Phase 2: Core Git Operations

Implementation of the Git service to query repositories without relying on heavy third-party git bindings. We use `os/exec` for performance and accuracy matching the user's terminal.

```go
// [NEW FILE] internal/git/git.go
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// RepoStatus holds the current state of a git repository
type RepoStatus struct {
	Name        string
	Path        string
	IsDirty     bool
	NeedsPull   bool
	NeedsPush   bool
	CurrentBranch string
	Error       error
}

// CheckStatus runs git commands to determine the repository state
func CheckStatus(name, path string) RepoStatus {
	status := RepoStatus{Name: name, Path: path}

	// Get current branch
	branchCmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	if out, err := branchCmd.Output(); err == nil {
		status.CurrentBranch = strings.TrimSpace(string(out))
	} else {
		status.Error = fmt.Errorf("not a valid git repo or no commits")
		return status
	}

	// Check if dirty (uncommitted changes)
	statusCmd := exec.Command("git", "-C", path, "status", "--porcelain")
	if out, err := statusCmd.Output(); err == nil {
		status.IsDirty = len(bytes.TrimSpace(out)) > 0
	} else {
		status.Error = err
		return status
	}

	// Fetch remote to check sync status reliably (can be slow, might need async handling later)
	exec.Command("git", "-C", path, "fetch").Run()

	// Check unpulled/unpushed commits
	revCmd := exec.Command("git", "-C", path, "rev-list", "--left-right", "--count", status.CurrentBranch+"...@{u}")
	if out, err := revCmd.Output(); err == nil {
		counts := strings.Fields(string(out))
		if len(counts) == 2 {
			status.NeedsPush = counts[0] != "0"
			status.NeedsPull = counts[1] != "0"
		}
	}

	return status
}

// Pull executes git pull on the specified repository
func Pull(path string) error {
	cmd := exec.Command("git", "-C", path, "pull", "--rebase")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}
	return nil
}
```

## Phase 3: Bubble Tea TUI Implementation

Building the tiling UI. We will use a main Model that splits the screen into a List pane (left) and a Details pane (right).

```go
//[NEW FILE] internal/tui/tui.go
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/yourusername/multi-repo-ai-orchestrator/internal/config"
	"github.com/yourusername/multi-repo-ai-orchestrator/internal/git"
)

// Styling definitions (Tiling)
var (
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)
	
	dirtyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // Red
	cleanStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Green
	syncStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
)

// Messages
type statusMsg git.RepoStatus
type refreshMsg struct{}

// item represents a list item
type item struct {
	status git.RepoStatus
}
func (i item) Title() string       { return i.status.Name }
func (i item) Description() string { 
	if i.status.Error != nil {
		return "Error: " + i.status.Error.Error()
	}
	state := "Clean"
	if i.status.IsDirty {
		state = "Dirty ⚠️"
	}
	return fmt.Sprintf("%s | Branch: %s", state, i.status.CurrentBranch)
}
func (i item) FilterValue() string { return i.status.Name }

type model struct {
	list     list.Model
	spinner  spinner.Model
	repos[]config.RepoConfig
	statuses map[string]git.RepoStatus
	width    int
	height   int
}

func NewModel(cfg *config.Config) tea.Model {
	items := make([]list.Item, 0)
	for _, r := range cfg.Repositories {
		items = append(items, item{status: git.RepoStatus{Name: r.Name, Path: r.Path}})
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "AI Watched Repositories"
	
	s := spinner.New()
	s.Spinner = spinner.Dot

	return model{
		list:     l,
		spinner:  s,
		repos:    cfg.Repositories,
		statuses: make(map[string]git.RepoStatus),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, triggerRefresh())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds[]tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "p":
			// Trigger Pull All action
			return m, triggerPullAll(m.repos)
		case "r":
			return m, triggerRefresh()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width/3, msg.Height-4) // Left pane takes 1/3

	case refreshMsg:
		for _, repo := range m.repos {
			cmds = append(cmds, checkRepoStatus(repo))
		}

	case statusMsg:
		m.statuses[msg.Name] = git.RepoStatus(msg)
		// Update list item
		items := m.list.Items()
		for i, it := range items {
			if it.(item).status.Name == msg.Name {
				items[i] = item{status: git.RepoStatus(msg)}
			}
		}
		m.list.SetItems(items)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Tiling: Join Left Pane (List) and Right Pane (Details)
	leftPane := paneStyle.Width(m.width/3).Height(m.height-2).Render(m.list.View())
	
	details := m.renderDetails()
	rightPane := paneStyle.Width(m.width - (m.width/3) - 4).Height(m.height-2).Render(details)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}

func (m model) renderDetails() string {
	selected := m.list.SelectedItem()
	if selected == nil {
		return "No repository selected."
	}
	
	it := selected.(item)
	status := it.status

	doc := strings.Builder{}
	doc.WriteString(fmt.Sprintf("# %s\n\n", status.Name))
	doc.WriteString(fmt.Sprintf("Path: %s\n\n", status.Path))
	
	if status.Error != nil {
		doc.WriteString(dirtyStyle.Render(fmt.Sprintf("Error: %v\n", status.Error)))
		return doc.String()
	}

	doc.WriteString(fmt.Sprintf("Branch: %s\n", status.CurrentBranch))
	
	if status.IsDirty {
		doc.WriteString(dirtyStyle.Render("Status: DIRTY (Uncommitted changes)\n"))
	} else {
		doc.WriteString(cleanStyle.Render("Status: CLEAN\n"))
	}

	if status.NeedsPull {
		doc.WriteString(syncStyle.Render("⬇️ Needs Pull\n"))
	}
	if status.NeedsPush {
		doc.WriteString(syncStyle.Render("⬆️ Needs Push\n"))
	}

	doc.WriteString("\n\nCommands:\n[r] Refresh\n[p] Pull All Repositories\n[enter] Open Terminal Here (Coming soon)\n[q] Quit")

	return doc.String()
}

// Commands
func checkRepoStatus(repo config.RepoConfig) tea.Cmd {
	return func() tea.Msg {
		return statusMsg(git.CheckStatus(repo.Name, repo.Path))
	}
}

func triggerRefresh() tea.Cmd {
	return func() tea.Msg {
		return refreshMsg{}
	}
}

func triggerPullAll(repos[]config.RepoConfig) tea.Cmd {
	return func() tea.Msg {
		for _, repo := range repos {
			git.Pull(repo.Path)
		}
		return refreshMsg{} // Refresh statuses after pulling
	}
}
```

## Phase 4: Integration & Subcommands

Wire up the TUI into the Cobra CLI structure.

```go
// [NEW FILE] cmd/dashboard.go
package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/multi-repo-ai-orchestrator/internal/config"
	"github.com/yourusername/multi-repo-ai-orchestrator/internal/tui"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Launch the TUI dashboard",
	Aliases:[]string{"dash", "ui"},
	RunE: func(cmd *cobra.Command, args[]string) error {
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
```

## Phase 5: Documentation & Future Workflow Hooks

Create user documentation and a sample configuration.

```markdown
// [NEW FILE] README.md
# Multi-Repo AI Orchestrator (MRAO)

A highly responsive TUI dashboard written in Go/Bubble Tea for monitoring local Git repositories touched by autonomous AI coding agents.

## Setup

1. Create a configuration file at `~/.config/mrao/config.yaml`:
```yaml
repositories:
  - name: mrao-core
    path: /home/user/projects/mrao
  - name: api-backend
    path: /home/user/projects/api-backend
```

2. Run the dashboard:
```bash
mrao dashboard
```

## Shortcuts
- `Up/Down` or `j/k`: Navigate repositories
- `r`: Force refresh git states
- `p`: Git pull all watched repositories (Rebase)
- `q`: Quit
```

---

## Phase 6: Report Generation
*(To be executed by the AI Agent upon completion of Phases 1-5)*

Create a report using the exact requested structure to document the successful implementation of the plan, detailing any deviations (e.g., changes to Lipgloss padding or layout calculations based on terminal constraints).

```yaml
---
filename: "_ai/backlog/reports/{YYMMDD_HHmm}__IMPLEMENTATION_REPORT__git-agent-tui.md"
title: "Report: Multi-Repo AI Git Dashboard (TUI)"
createdAt: YYYY-MM-DD HH:mm
updatedAt: YYYY-MM-DD HH:mm
planFile: "_ai/backlog/active/{YYMMDD_HHmm}__IMPLEMENTATION_PLAN__git-agent-tui.md"
project: "multi-repo-ai-orchestrator"
status: completed
filesCreated: 0
filesModified: 0
filesDeleted: 0
tags:[golang, cli, tui, bubbletea, git, automation]
documentType: IMPLEMENTATION_REPORT
---
```
