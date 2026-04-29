package agenttab

type AgentDef struct {
	Command  string   `yaml:"command"`
	Args     []string `yaml:"args"`
	ModelArg string   `yaml:"model_arg"`
}

type FileConfig struct {
	WorktreesDir string              `yaml:"worktrees_dir"`
	ResultsFile  string              `yaml:"results_file"`
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
	resultsFile  string
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
	agents  []agentSpec
	session string
	prompt  string
	dryRun  bool
}

type agentSpec struct {
	Name  string
	Model string
	Label string
}

type candidate struct {
	agent    string
	model    string
	label    string
	codename string
	cmd      string
	path     string
	branch   string
	pane     string
}
