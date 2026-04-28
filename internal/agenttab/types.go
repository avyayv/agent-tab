package agenttab

type AgentDef struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type FileConfig struct {
	WorktreesDir string              `yaml:"worktrees_dir"`
	Shell        string              `yaml:"shell"`
	Judge        JudgeConfig         `yaml:"judge"`
	Tmux         TmuxConfig          `yaml:"tmux"`
	Agents       map[string]AgentDef `yaml:"agents"`
}

type JudgeConfig struct {
	Agent string `yaml:"agent"`
}

type TmuxConfig struct {
	Attach     *bool  `yaml:"attach"`
	AttachMode string `yaml:"attach_mode"`
	Layout     string `yaml:"layout"`
}

type cliOptions struct {
	configPath   string
	worktreesDir string
	judge        string
	session      string
	layout       string
	attachMode   string
	attachSet    bool
	attach       bool
	dryRun       bool
	agentsFlag   string
	showConfig   bool
	positionals  []string
	prompt       string
}

type config struct {
	file    FileConfig
	agents  []string
	session string
	prompt  string
	dryRun  bool
}

type candidate struct {
	agent  string
	cmd    string
	path   string
	branch string
	pane   string
}
