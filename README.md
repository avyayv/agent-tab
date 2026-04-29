# agent-tab

`agent-tab` runs coding-agent A/B tests in isolated git worktrees.

It creates one temporary worktree per contestant, opens the contestants in tmux, and starts a judge agent in the base worktree. The judge is told to wait until you say the contestants are done before comparing results.

## Install

```bash
go install github.com/avyayv/agent-tab/cmd/agent-tab@latest
```

From a local checkout:

```bash
go install ./cmd/agent-tab
```

## Usage

Run from inside a git repository:

```bash
agent-tab                                      # codex + pi
agent-tab codex claude -- "implement X"
agent-tab codex/gpt-5.5 claude/claude-opus-4.7 -- "implement X"
agent-tab all -- "implement X"                # first three configured agents
agent-tab pi claude my-ab -- "implement X"    # custom tmux session name
```

## What it does

- Creates one fresh, anonymized worktree per contestant under `~/.agent-tab/worktrees` by default.
- Copies tracked local changes and untracked non-ignored files into each contestant worktree.
- Symlinks `.env*` files and `node_modules` directories from the base worktree.
- Opens contestant agents in a tmux tab/window.
- Opens the judge agent in the base/current worktree.
- Sends the task prompt to contestants immediately.
- Sends the judge a coordinator prompt with anonymized candidate codenames and worktree paths.
- Never cleans up worktrees automatically.

## Configuration

Config is loaded from `~/.config/agent-tab/config.yaml` by default. Override it with `--config` or `AGENT_TAB_CONFIG`.

```yaml
worktrees_dir: ~/.agent-tab/worktrees
results_file: ~/.agent-tab/results.json
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
    model_arg: --model
  claude:
    command: claude
    args: ["--dangerously-skip-permissions"]
    model_arg: --model
  pi:
    command: pi
    args: []
    model_arg: --model
```

Precedence is: flags, then environment variables, then config file, then defaults.

Environment variables:

```bash
AGENT_TAB_CONFIG
AGENT_TAB_WORKTREES_DIR
AGENT_TAB_RESULTS_FILE
AGENT_TAB_ATTACH_MODE
AGENT_TAB_JUDGE
AGENT_TAB_LAYOUT
```

## Flags

```bash
agent-tab [flags] [all|agent[/model]...] [session_name] [-- prompt]

--config PATH
--worktrees-dir PATH
--results-file PATH
--judge AGENT
--session NAME
--agents a,b[,c]       agents may include /model
--layout tiled|even-horizontal|even-vertical
--attach-mode normal|iterm-control-mode|none
--attach / --no-attach
--dry-run
--show-config
```

## Tracking results

Record a completed A/B test with an ordered ranking:

```bash
agent-tab record --task-type frontend --order codex,claude,pi --notes "codex was simplest"
```

Results are appended to `~/.agent-tab/results.json` by default:

```json
{
  "version": 1,
  "runs": [
    {
      "id": "20260428-131358-49287",
      "timestamp": "2026-04-28T20:13:58Z",
      "repo": "/path/to/repo",
      "branch": "main",
      "task_type": "frontend",
      "order": ["codex", "claude", "pi"],
      "notes": "codex was simplest"
    }
  ]
}
```

View aggregate stats:

```bash
agent-tab stats
agent-tab stats --task-type frontend
```

Use `--results-file PATH` or `AGENT_TAB_RESULTS_FILE` to store results somewhere else.

## Safety

`agent-tab` does not delete worktrees or branches. The judge prompt explicitly says to wait until you say contestants are done, ask before applying a winner, and ask separately before cleanup.
