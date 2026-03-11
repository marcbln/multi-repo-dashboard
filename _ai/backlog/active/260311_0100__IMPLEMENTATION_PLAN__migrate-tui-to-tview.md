---
filename: "_ai/backlog/active/260311_0100__IMPLEMENTATION_PLAN__migrate-tui-to-tview.md"
title: "Migrate TUI from Bubbletea to Tview"
createdAt: 2026-03-11 01:00
createdBy: Cascade [Cascade]
updatedAt: 2026-03-11 01:00
updatedBy: Cascade [Cascade]
status: draft
priority: high
tags: [tui, migration, tview, bubbletea]
estimatedComplexity: moderate
documentType: IMPLEMENTATION_PLAN
---

## Problem Statement
The current Multi-Repo Dashboard utilizes the Bubbletea library (along with Bubbles and Lipgloss) for its Terminal User Interface (TUI). To build a highly dense, feature-rich dashboard with tables, modals, and intricate layouts—similar to K9s—we need to migrate the project from Bubbletea to `tview`. The `tview` library uses a Widget-based, Object-Oriented approach that natively supports these elements without having to build them from scratch.

## Implementation Notes
- **Context Details**: The project is a Go application to manage git repositories. The core business logic (`internal/git`) and configuration logic (`internal/config`) will remain completely untouched. Only the UI layer needs modification.
- **Root Directory**: `/home/marc/devel/multi-repo-dashboard`
- **TUI Logic**: Primarily contained in `internal/tui/tui.go` and invoked in `cmd/dashboard.go`.
- **Command to Run**: `go run main.go dashboard` (or your binary name `mrd dashboard`).
- **Concurrency Consideration**: `tview` uses mutable state, so background processes like checking git statuses and pulling repositories must be executed in goroutines, and any UI updates from those goroutines must be wrapped in `app.QueueUpdateDraw(func() { ... })` to ensure thread safety and avoid race conditions.

## Phase 1: Update Dependencies
**Objective**: Remove Bubbletea packages and install tview.
**Tasks**:
- Remove `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles`, `github.com/charmbracelet/lipgloss`.
- Add `github.com/rivo/tview` and `github.com/gdamore/tcell/v2`.
- Run `go mod tidy`.

## Phase 2: Refactor TUI Package
**Objective**: Rewrite `internal/tui/tui.go` to use `tview` components.
**Tasks**:
- Implement a struct wrapping `tview.Application` and key widgets (`tview.List`, `tview.TextView`, `tview.Flex`).
- Set up a split-pane layout: a List on the left (1/3 width) for repositories, and a TextView on the right for repository details and logs.
- Port over keybindings via `SetInputCapture`:
  - `q` or `Ctrl+C` to quit.
  - `r` to refresh the current/all repos.
  - `p` to pull the current/all repos.
- Implement background goroutines for Git operations (`git.CheckStatus`, `git.Pull`), updating UI components using `app.QueueUpdateDraw()`.

**Code Snippet Example**:
```go [MODIFY] internal/tui/tui.go
package tui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/yourusername/multi-repo-dashboard/internal/config"
	// other imports...
)

// Define your TUI application structure and logic here
// ...
```

## Phase 3: Update Command Entrypoint
**Objective**: Modify `cmd/dashboard.go` to initialize and invoke the new TUI application.
**Tasks**:
- Replace `tea.NewProgram(...)` and its setup with the initialisation of the new `tview` based interface.

**Code Snippet Example**:
```go [MODIFY] cmd/dashboard.go
// Remove Bubbletea references
// Initialize tui app and run it
// e.g., app := tui.NewApp(&cfg); err := app.Run()
```

## Phase 4: Maintenance & Reporting
**Objective**: Verify the application runs correctly and write the final implementation report.
**Tasks**:
- Verify that `go run main.go dashboard` opens the new UI.
- Verify that selecting repositories displays their info.
- Verify background fetches (`r`) and pulls (`p`) update the interface without crashing or freezing.
- Generate the final report document.

**Deliverables**:
Write a comprehensive report to the following path:
`_ai/backlog/reports/260311_0100__IMPLEMENTATION_REPORT__migrate-tui-to-tview.md`

### Required Report Format
```yaml
---
filename: "_ai/backlog/reports/260311_0100__IMPLEMENTATION_REPORT__migrate-tui-to-tview.md"
title: "Report: Migrate TUI from Bubbletea to Tview"
createdAt: 2026-03-11 01:00
createdBy: Cascade [Cascade]
updatedAt: 2026-03-11 01:00
updatedBy: Cascade [Cascade]
planFile: "_ai/backlog/active/260311_0100__IMPLEMENTATION_PLAN__migrate-tui-to-tview.md"
project: "multi-repo-dashboard"
status: completed
filesCreated: 0
filesModified: 3
filesDeleted: 0
tags: [tui, migration, tview, bubbletea]
documentType: IMPLEMENTATION_REPORT
---
```
Include Summary, Files Changed, Key Changes, Technical Decisions, Testing Notes, and Next Steps in the report content.
