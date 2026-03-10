# Multi-Repo AI Orchestrator (mrd)

A highly responsive TUI dashboard written in Go/Bubble Tea for monitoring local Git repositories touched by autonomous AI coding agents.

## Setup

1. Create a configuration file at `~/.config/mrd/config.yaml`:
```yaml
repositories:
  - name: mrd-core
    path: /home/user/projects/mrd
  - name: api-backend
    path: /home/user/projects/api-backend
```

2. Run the dashboard:
```bash
mrd dashboard
```

## Shortcuts
- `Up/Down` or `j/k`: Navigate repositories
- `r`: Force refresh git states
- `p`: Git pull all watched repositories (Rebase)
- `q`: Quit
