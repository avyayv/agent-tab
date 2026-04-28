# agent-tab

`agenttab` runs coding-agent A/B tests in isolated git worktrees.

It creates one temporary worktree per contestant, opens the contestants in tmux, and starts a judge agent in the base worktree. The judge is told to wait until you say the contestants are done before comparing results.

## Install

```bash
go install github.com/avyayv/agent-tab/cmd/agenttab@latest
```

From a local checkout:

```bash
go install ./cmd/agenttab
```

## Usage

Run from inside a git repository:

```bash
agenttab                                      # codex + pi
agenttab codex claude -- "implement X"
agenttab all -- "implement X"                # first three configured agents
agenttab pi claude my-ab -- "implement X"    # custom tmux session name
```

## What it does

- Creates one fresh worktree per contestant under `~/.agenttab/worktrees` by default.
- Copies tracked local changes and untracked non-ignored files into each contestant worktree.
- Symlinks `.env*` files and `node_modules` directories from the base worktree.
- Opens contestant agents in a tmux tab/window.
- Opens the judge agent in the base/current worktree.
- Sends the task prompt to contestants immediately.
- Sends the judge a coordinator prompt with all candidate worktree paths.
- Never cleans up worktrees automatically.

## Configuration

Config is loaded from `~/.config/agenttab/config.yaml` by default. Override it with `--config` or `AGENTTAB_CONFIG`.

```yaml
worktrees_dir: ~/.agenttab/worktrees
shell: /bin/zsh

judge:
  agent: pi

tmux:
  attach: true
  attach_mode: normal # normal | iterm-control-mode | none
  layout: tiled

agents:
  codex:
    command: codex
    args: ["--yolo"]
  claude:
    command: claude
    args: ["--yolo"]
  pi:
    command: pi
    args: []
```

Precedence is: flags, then environment variables, then config file, then defaults.

Environment variables:

```bash
AGENTTAB_CONFIG
AGENTTAB_WORKTREES_DIR
AGENTTAB_ATTACH_MODE
AGENTTAB_JUDGE
AGENTTAB_LAYOUT
```

## Flags

```bash
agenttab [flags] [all|agent...] [session_name] [-- prompt]

--config PATH
--worktrees-dir PATH
--judge AGENT
--session NAME
--agents a,b[,c]
--layout tiled|even-horizontal|even-vertical
--attach-mode normal|iterm-control-mode|none
--attach / --no-attach
--dry-run
--show-config
```

## Safety

`agenttab` does not delete worktrees or branches. The judge prompt explicitly says to wait until you say contestants are done, ask before applying a winner, and ask separately before cleanup.
