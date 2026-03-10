package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/multi-repo-dashboard/internal/config"
	"github.com/yourusername/multi-repo-dashboard/internal/git"
)

// Styling definitions (Tiling)
var (
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)

	dirtyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // Red
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

func (i item) Title() string { return i.status.Name }
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
	repos    []config.RepoConfig
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
	var cmds []tea.Cmd

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
	leftPane := paneStyle.Width(m.width/3).Height(m.height - 2).Render(m.list.View())

	details := m.renderDetails()
	rightPane := paneStyle.Width(m.width - (m.width / 3) - 4).Height(m.height - 2).Render(details)

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

func triggerPullAll(repos []config.RepoConfig) tea.Cmd {
	return func() tea.Msg {
		for _, repo := range repos {
			git.Pull(repo.Path)
		}
		return refreshMsg{} // Refresh statuses after pulling
	}
}
