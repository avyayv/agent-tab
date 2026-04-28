# agenttab

`agenttab` runs coding-agent A/B tests in isolated git worktrees. It creates one temporary worktree per contestant, opens the contestants in tmux, and starts a judge agent in the base worktree.

```bash
go install ./cmd/agenttab

agenttab                                      # codex + pi
agenttab codex claude -- "implement X"
agenttab all -- "implement X"                # first three configured agents
agenttab pi claude my-ab -- "implement X"    # custom tmux session name
```

By default, worktrees are created under `~/.agenttab/worktrees` and tmux attaches normally. iTerm control mode is opt-in.

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
  layout: tiled       # any tmux layout, e.g. tiled or even-horizontal

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

Supported environment variables:

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

`agenttab` never cleans up worktrees automatically. The judge prompt explicitly tells the judge to wait until you say contestants are done, then ask before applying a winner, and ask separately before cleanup.
