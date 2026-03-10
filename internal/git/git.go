package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// RepoStatus holds the current state of a git repository
type RepoStatus struct {
	Name          string
	Path          string
	IsDirty       bool
	NeedsPull     bool
	NeedsPush     bool
	CurrentBranch string
	Error         error
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
