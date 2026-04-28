package agenttab

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"sort"
)

func defaultConfig() FileConfig {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return FileConfig{
		WorktreesDir: "~/.agenttab/worktrees",
		Shell:        shell,
		Judge:        JudgeConfig{Agent: "pi"},
		Tmux: TmuxConfig{
			Attach:     boolPtr(true),
			AttachMode: "normal",
			Layout:     "tiled",
		},
		Agents: map[string]AgentDef{
			"codex":  {Command: "codex", Args: []string{"--yolo"}},
			"claude": {Command: "claude", Args: []string{"--yolo"}},
			"pi":     {Command: "pi"},
		},
	}
}

func boolPtr(v bool) *bool { return &v }

func attachEnabled(fc FileConfig) bool {
	if fc.Tmux.Attach == nil {
		return true
	}
	return *fc.Tmux.Attach
}

func loadConfigFile(fc *FileConfig, path string) error {
	if path == "" {
		path = defaultConfigPath()
	}
	expanded, err := expandPath(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(expanded)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var override FileConfig
	if err := yaml.Unmarshal(data, &override); err != nil {
		return fmt.Errorf("read config %s: %w", expanded, err)
	}
	mergeConfig(fc, override)
	return nil
}

func mergeConfig(dst *FileConfig, src FileConfig) {
	if src.WorktreesDir != "" {
		dst.WorktreesDir = src.WorktreesDir
	}
	if src.Shell != "" {
		dst.Shell = src.Shell
	}
	if src.Judge.Agent != "" {
		dst.Judge.Agent = src.Judge.Agent
	}
	if src.Tmux.AttachMode != "" {
		dst.Tmux.AttachMode = src.Tmux.AttachMode
	}
	if src.Tmux.Layout != "" {
		dst.Tmux.Layout = src.Tmux.Layout
	}
	if src.Tmux.Attach != nil {
		dst.Tmux.Attach = src.Tmux.Attach
	}
	if src.Agents != nil {
		if dst.Agents == nil {
			dst.Agents = map[string]AgentDef{}
		}
		for name, def := range src.Agents {
			dst.Agents[name] = def
		}
	}
}

func applyEnv(fc *FileConfig) {
	if v := os.Getenv("AGENTTAB_WORKTREES_DIR"); v != "" {
		fc.WorktreesDir = v
	}
	if v := os.Getenv("AGENTTAB_ATTACH_MODE"); v != "" {
		fc.Tmux.AttachMode = v
	}
	if v := os.Getenv("AGENTTAB_JUDGE"); v != "" {
		fc.Judge.Agent = v
	}
	if v := os.Getenv("AGENTTAB_LAYOUT"); v != "" {
		fc.Tmux.Layout = v
	}
}

func applyFlags(fc *FileConfig, opts cliOptions) {
	if opts.worktreesDir != "" {
		fc.WorktreesDir = opts.worktreesDir
	}
	if opts.judge != "" {
		fc.Judge.Agent = opts.judge
	}
	if opts.layout != "" {
		fc.Tmux.Layout = opts.layout
	}
	if opts.attachMode != "" {
		fc.Tmux.AttachMode = opts.attachMode
	}
	if opts.attachSet {
		fc.Tmux.Attach = boolPtr(opts.attach)
		if !opts.attach {
			fc.Tmux.AttachMode = "none"
		}
	}
	if fc.Tmux.AttachMode == "none" {
		fc.Tmux.Attach = boolPtr(false)
	}
}

func configuredAgentNames(fc FileConfig) []string {
	names := make([]string, 0, len(fc.Agents))
	preferred := []string{"codex", "claude", "pi"}
	seen := map[string]bool{}
	for _, p := range preferred {
		if _, ok := fc.Agents[p]; ok {
			names = append(names, p)
			seen[p] = true
		}
	}
	other := []string{}
	for name := range fc.Agents {
		if !seen[name] {
			other = append(other, name)
		}
	}
	sort.Strings(other)
	names = append(names, other...)
	if len(names) > 3 {
		return names[:3]
	}
	return names
}
