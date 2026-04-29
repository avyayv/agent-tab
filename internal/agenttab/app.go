package agenttab

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func Run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "record":
			return recordCommand(args[1:])
		case "stats":
			return statsCommand(args[1:])
		}
	}

	opts, err := parseCLI(args)
	if err != nil {
		return err
	}
	if opts.configPath == "" {
		opts.configPath = os.Getenv("AGENT_TAB_CONFIG")
	}

	fc := defaultConfig()
	if err := loadConfigFile(&fc, opts.configPath); err != nil {
		return err
	}
	applyEnv(&fc)
	applyFlags(&fc, opts)

	if opts.showConfig {
		enc := yaml.NewEncoder(os.Stdout)
		enc.SetIndent(2)
		defer enc.Close()
		return enc.Encode(fc)
	}

	cfg, err := buildRunConfig(fc, opts)
	if err != nil {
		return err
	}

	commands := []string{"git", "tmux"}
	for _, agent := range cfg.agents {
		def := fc.Agents[agent.Name]
		commands = append(commands, def.Command)
	}
	judgeDef, ok := fc.Agents[fc.Judge.Agent]
	if !ok {
		return fmt.Errorf("judge agent %q is not configured", fc.Judge.Agent)
	}
	commands = append(commands, judgeDef.Command)
	if cfg.dryRun {
		commands = []string{"git"}
	}
	if err := requireCommands(commands); err != nil {
		return err
	}

	sourceDir, err := output("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return errors.New("agent-tab must be run inside a git repository")
	}
	sourceDir = strings.TrimSpace(sourceDir)
	repoName := filepath.Base(sourceDir)
	currentRef, _ := outputIn(sourceDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	currentRef = strings.TrimSpace(currentRef)
	if currentRef == "HEAD" || currentRef == "" {
		currentRef, _ = outputIn(sourceDir, "git", "rev-parse", "--short", "HEAD")
		currentRef = strings.TrimSpace(currentRef)
	}
	safeRef := sanitize(currentRef)
	stamp := fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), os.Getpid())

	wtBase, err := expandPath(fc.WorktreesDir)
	if err != nil {
		return err
	}
	if cfg.dryRun {
		resultsFile, _ := expandPath(fc.ResultsFile)
		fmt.Printf("worktrees_dir: %s\n", wtBase)
		fmt.Printf("results_file: %s\n", resultsFile)
		fmt.Printf("judge: %s (%s)\n", fc.Judge.Agent, commandLine(judgeDef))
		fmt.Printf("attach_mode: %s\n", fc.Tmux.AttachMode)
		fmt.Printf("layout: %s\n", fc.Tmux.Layout)
		fmt.Println("candidates:")
		for i, agent := range cfg.agents {
			codename := codenameForIndex(i)
			path := filepath.Join(wtBase, fmt.Sprintf("%s-%s-agent-tab-%s-%s", repoName, safeRef, codename, stamp))
			branch := fmt.Sprintf("agent-tab/%s/%s-%s", safeRef, codename, stamp)
			fmt.Printf("  - %s (%s): %s (%s) command=%s\n", codename, agent.Label, path, branch, commandLineForSpec(fc.Agents[agent.Name], agent))
		}
		return nil
	}
	if err := os.MkdirAll(wtBase, 0o755); err != nil {
		return err
	}

	patchFile, cleanupPatch, err := makePatch(sourceDir)
	if err != nil {
		return err
	}
	defer cleanupPatch()

	candidates := make([]candidate, 0, len(cfg.agents))
	fmt.Println("Creating anonymized worktrees:")
	for i, agent := range cfg.agents {
		codename := codenameForIndex(i)
		cand := candidate{
			agent:    agent.Name,
			model:    agent.Model,
			label:    agent.Label,
			codename: codename,
			cmd:      commandLineForSpec(fc.Agents[agent.Name], agent),
			path:     filepath.Join(wtBase, fmt.Sprintf("%s-%s-agent-tab-%s-%s", repoName, safeRef, codename, stamp)),
			branch:   fmt.Sprintf("agent-tab/%s/%s-%s", safeRef, codename, stamp),
		}
		fmt.Printf("  %s -> %s (%s)\n", cand.codename, cand.path, cand.label)
		if err := commandIn(sourceDir, "git", "worktree", "add", "-b", cand.branch, cand.path, "HEAD").Run(); err != nil {
			return fmt.Errorf("failed to create %s; any already-created worktrees were left in place for manual review: %w", cand.path, err)
		}
		candidates = append(candidates, cand)
	}

	for _, cand := range candidates {
		if err := copyWorktreeContext(sourceDir, cand.path, patchFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not copy all local context into %s: %v\n", cand.path, err)
		}
	}

	judgePrompt := buildJudgePrompt(cfg.prompt, candidates)
	if os.Getenv("TMUX") != "" {
		return openInsideTmux(sourceDir, cfg, candidates, judgePrompt)
	}
	return openNewTmuxSession(sourceDir, cfg, candidates, judgePrompt)
}
