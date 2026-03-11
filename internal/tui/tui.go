package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yourusername/multi-repo-dashboard/internal/config"
	"github.com/yourusername/multi-repo-dashboard/internal/git"
)

// App represents the TUI application
type App struct {
	app         *tview.Application
	repoList    *tview.List
	detailsView *tview.TextView
	flex        *tview.Flex
	repos       []config.RepoConfig
	statuses    map[string]git.RepoStatus
	isPulling   bool
	isRefreshing bool
	logs        string
	toastMsg    string
	toastTimer  *time.Timer
}

// NewApp creates a new TUI application
func NewApp(cfg *config.Config) *App {
	app := tview.NewApplication()
	
	// Create repository list (left pane)
	repoList := tview.NewList()
	repoList.ShowSecondaryText(true)
	repoList.SetBorder(true)
	repoList.SetTitle("Watched Repositories")
	repoList.SetHighlightFullLine(true)
	repoList.SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack))
	
	// Create details view (right pane)
	detailsView := tview.NewTextView()
	detailsView.SetDynamicColors(true)
	detailsView.SetBorder(true)
	detailsView.SetTitle("Repository Details")
	detailsView.SetScrollable(true)
	
	// Create main flex layout
	flex := tview.NewFlex()
	flex.AddItem(repoList, 0, 1, true)  // Left pane takes 1/3
	flex.AddItem(detailsView, 0, 2, false) // Right pane takes 2/3
	
	tuiApp := &App{
		app:         app,
		repoList:    repoList,
		detailsView: detailsView,
		flex:        flex,
		repos:       cfg.Repositories,
		statuses:    make(map[string]git.RepoStatus),
	}
	
	// Set up input capture for global keybindings
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC, tcell.KeyEscape:
			if event.Rune() == 'q' || event.Key() == tcell.KeyCtrlC || event.Key() == tcell.KeyEscape {
				app.Stop()
				return nil
			}
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q':
				app.Stop()
				return nil
			case 'r':
				if !tuiApp.isPulling && !tuiApp.isRefreshing {
					tuiApp.refreshAll()
				}
				return nil
			case 'p':
				if !tuiApp.isPulling && !tuiApp.isRefreshing {
					tuiApp.pullAll()
				}
				return nil
			}
		}
		return event
	})
	
	// Set up list selection handler
	repoList.SetSelectedFunc(func(index int, name string, secondary string, shortcut rune) {
		tuiApp.updateDetails(index)
	})
	
	// Populate initial repository list
	for _, repo := range cfg.Repositories {
		repoList.AddItem(repo.Name, "Loading...", 'r', nil)
	}
	
	// Start initial refresh
	tuiApp.refreshAll()
	
	return tuiApp
}

// Run starts the TUI application
func (a *App) Run() error {
	return a.app.SetRoot(a.flex, true).EnableMouse(true).Run()
}

// refreshAll triggers status refresh for all repositories
func (a *App) refreshAll() {
	a.isRefreshing = true
	a.showToast("Refreshing repositories...")
	
	// Run refresh in background goroutine
	go func() {
		defer func() {
			a.app.QueueUpdateDraw(func() {
				a.isRefreshing = false
			})
		}()
		
		for _, repo := range a.repos {
			status := git.CheckStatus(repo.Name, repo.Path)
			a.statuses[repo.Name] = status
			
			// Update UI in main thread
			a.app.QueueUpdateDraw(func() {
				a.updateRepoItem(repo.Name, status)
				selectedIndex := a.repoList.GetCurrentItem()
				if selectedIndex >= 0 && selectedIndex < len(a.repos) {
					if a.repos[selectedIndex].Name == repo.Name {
						a.updateDetails(selectedIndex)
					}
				}
			})
		}
		
		// Show completion message
		a.app.QueueUpdateDraw(func() {
			a.showToast("Repositories refreshed successfully!")
		})
	}()
}

// pullAll triggers pull for all repositories
func (a *App) pullAll() {
	a.isPulling = true
	a.logs = ""
	a.showToast("Pulling all repositories...")
	
	// Run pull in background goroutine
	go func() {
		defer func() {
			a.app.QueueUpdateDraw(func() {
				a.isPulling = false
			})
		}()
		
		var allLogs strings.Builder
		var lastErr error
		
		for _, repo := range a.repos {
			allLogs.WriteString(fmt.Sprintf("--- Pulling %s ---\n", repo.Name))
			logs, err := git.Pull(repo.Path)
			allLogs.WriteString(logs)
			allLogs.WriteString("\n")
			if err != nil {
				lastErr = err
			}
		}
		
		// Update UI in main thread
		a.app.QueueUpdateDraw(func() {
			a.logs = allLogs.String()
			if lastErr != nil {
				a.showToast(fmt.Sprintf("Pull failed: %v", lastErr))
			} else {
				a.showToast("All repositories pulled successfully!")
			}
			
			// Refresh to show updated status
			go a.refreshAll()
			
			// Update details view if a repo is selected
			selectedIndex := a.repoList.GetCurrentItem()
			if selectedIndex >= 0 {
				a.updateDetails(selectedIndex)
			}
		})
	}()
}

// updateRepoItem updates the list item for a repository
func (a *App) updateRepoItem(repoName string, status git.RepoStatus) {
	// Find the index of this repo in the list
	for i, repo := range a.repos {
		if repo.Name == repoName {
			secondaryText := a.formatRepoStatus(status)
			a.repoList.SetItemText(i, repoName, secondaryText)
			break
		}
	}
}

// formatRepoStatus formats repository status for display
func (a *App) formatRepoStatus(status git.RepoStatus) string {
	if status.Error != nil {
		return fmt.Sprintf("[red]Error: %s[-]", status.Error.Error())
	}
	
	state := "[green]Clean[-]"
	if status.IsDirty {
		state = "[red]Dirty ⚠️[-]"
	}
	
	secondary := fmt.Sprintf("%s | Branch: %s", state, status.CurrentBranch)
	
	if status.NeedsPull {
		secondary += " | [yellow]⬇️ Needs Pull[-]"
	}
	if status.NeedsPush {
		secondary += " | [yellow]⬆️ Needs Push[-]"
	}
	
	return secondary
}

// updateDetails updates the details view for the selected repository
func (a *App) updateDetails(index int) {
	if index < 0 || index >= len(a.repos) {
		a.detailsView.SetText("No repository selected.")
		return
	}
	
	repo := a.repos[index]
	status, exists := a.statuses[repo.Name]
	if !exists {
		a.detailsView.SetText(fmt.Sprintf("Loading status for %s...", repo.Name))
		return
	}
	
	content := a.formatDetailsContent(status)
	a.detailsView.SetText(content)
}

// formatDetailsContent formats the detailed content for a repository
func (a *App) formatDetailsContent(status git.RepoStatus) string {
	var content strings.Builder
	
	// Header with loading indicator
	header := fmt.Sprintf("[#ff0000::b]%s[-]", status.Name)
	if a.isPulling {
		header += " [yellow]Pulling...[-]"
	} else if a.isRefreshing {
		header += " [yellow]Refreshing...[-]"
	}
	content.WriteString(header)
	content.WriteString("\n\n")
	
	content.WriteString(fmt.Sprintf("Path: %s\n\n", status.Path))
	
	if status.Error != nil {
		content.WriteString(fmt.Sprintf("[red]Error: %v[-]\n", status.Error))
		return content.String()
	}
	
	content.WriteString(fmt.Sprintf("Branch: %s\n", status.CurrentBranch))
	
	if status.IsDirty {
		content.WriteString("[red]Status: DIRTY (Uncommitted changes)[-]\n")
	} else {
		content.WriteString("[green]Status: CLEAN[-]\n")
	}
	
	if status.NeedsPull {
		content.WriteString("[yellow]⬇️ Needs Pull[-]\n")
	}
	if status.NeedsPush {
		content.WriteString("[yellow]⬆️ Needs Push[-]\n")
	}
	
	content.WriteString("\n\nCommands:\n[r] Refresh\n[p] Pull All Repositories\n[enter] Open Terminal Here (Coming soon)\n[q] Quit")
	
	if a.logs != "" {
		content.WriteString("\n\nLogs:\n")
		content.WriteString("[gray]")
		content.WriteString(a.logs)
		content.WriteString("[-]")
	}
	
	return content.String()
}

// showToast displays a temporary toast message
func (a *App) showToast(message string) {
	a.toastMsg = message
	
	// Clear existing timer
	if a.toastTimer != nil {
		a.toastTimer.Stop()
	}
	
	// Set new timer to clear toast after 3 seconds
	a.toastTimer = time.AfterFunc(3*time.Second, func() {
		a.app.QueueUpdateDraw(func() {
			a.toastMsg = ""
		})
	})
}
