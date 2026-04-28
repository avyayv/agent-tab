package agenttab

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func openInsideTmux(sourceDir string, cfg config, candidates []candidate, judgePrompt string) error {
	session, err := output("tmux", "display-message", "-p", "#S")
	if err != nil {
		return err
	}
	session = strings.TrimSpace(session)
	currentPane, _ := output("tmux", "display-message", "-p", "#{pane_id}")
	currentPane = strings.TrimSpace(currentPane)

	firstPane, err := output("tmux", "new-window", "-P", "-F", "#{pane_id}", "-t", session+":", "-n", "ab-test", "-c", candidates[0].path, shellCmd(cfg.file, candidates[0].cmd))
	if err != nil {
		return err
	}
	candidates[0].pane = strings.TrimSpace(firstPane)
	windowIndex, _ := output("tmux", "display-message", "-p", "-t", candidates[0].pane, "#I")
	windowTarget := session + ":" + strings.TrimSpace(windowIndex)

	secondPane, err := output("tmux", "split-window", "-P", "-F", "#{pane_id}", "-h", "-t", windowTarget, "-c", candidates[1].path, shellCmd(cfg.file, candidates[1].cmd))
	if err != nil {
		return err
	}
	candidates[1].pane = strings.TrimSpace(secondPane)
	if len(candidates) == 3 {
		thirdPane, err := output("tmux", "split-window", "-P", "-F", "#{pane_id}", "-v", "-t", candidates[1].pane, "-c", candidates[2].path, shellCmd(cfg.file, candidates[2].cmd))
		if err != nil {
			return err
		}
		candidates[2].pane = strings.TrimSpace(thirdPane)
	}
	_ = command("tmux", "select-layout", "-t", windowTarget, cfg.file.Tmux.Layout).Run()
	for _, cand := range candidates {
		sendPrompt(cand.pane, cfg.prompt)
	}
	sendPrompt(currentPane, judgePrompt)
	fmt.Printf("Opened %s contestants. Starting %s judge here.\n", strings.Join(cfg.agents, ", "), cfg.file.Judge.Agent)
	judgeCmd := commandLine(cfg.file.Agents[cfg.file.Judge.Agent])
	return syscall.Exec(findExecutable(cfg.file.Shell), []string{cfg.file.Shell, "-lc", judgeCmd}, os.Environ())
}

func openNewTmuxSession(sourceDir string, cfg config, candidates []candidate, judgePrompt string) error {
	base := cfg.session
	if base == "" {
		base = "agenttab-ab-test"
	}
	session := base
	for i := 2; command("tmux", "has-session", "-t", session).Run() == nil; i++ {
		session = fmt.Sprintf("%s-%d", base, i)
	}
	judgeCmd := commandLine(cfg.file.Agents[cfg.file.Judge.Agent])
	if err := command("tmux", "new-session", "-d", "-s", session, "-n", "judge", "-c", sourceDir, shellCmd(cfg.file, judgeCmd)).Run(); err != nil {
		return err
	}
	judgePane, _ := output("tmux", "list-panes", "-t", session+":", "-F", "#{pane_id}")
	judgePane = strings.TrimSpace(strings.Split(judgePane, "\n")[0])
	judgeWindow, _ := output("tmux", "display-message", "-p", "-t", judgePane, "#I")
	judgeWindow = strings.TrimSpace(judgeWindow)

	firstPane, err := output("tmux", "new-window", "-P", "-F", "#{pane_id}", "-t", session+":", "-n", "ab-test", "-c", candidates[0].path, shellCmd(cfg.file, candidates[0].cmd))
	if err != nil {
		return err
	}
	candidates[0].pane = strings.TrimSpace(firstPane)
	windowIndex, _ := output("tmux", "display-message", "-p", "-t", candidates[0].pane, "#I")
	windowTarget := session + ":" + strings.TrimSpace(windowIndex)

	secondPane, err := output("tmux", "split-window", "-P", "-F", "#{pane_id}", "-h", "-t", windowTarget, "-c", candidates[1].path, shellCmd(cfg.file, candidates[1].cmd))
	if err != nil {
		return err
	}
	candidates[1].pane = strings.TrimSpace(secondPane)
	if len(candidates) == 3 {
		thirdPane, err := output("tmux", "split-window", "-P", "-F", "#{pane_id}", "-v", "-t", candidates[1].pane, "-c", candidates[2].path, shellCmd(cfg.file, candidates[2].cmd))
		if err != nil {
			return err
		}
		candidates[2].pane = strings.TrimSpace(thirdPane)
	}
	_ = command("tmux", "select-layout", "-t", windowTarget, cfg.file.Tmux.Layout).Run()
	for _, cand := range candidates {
		sendPrompt(cand.pane, cfg.prompt)
	}
	sendPrompt(judgePane, judgePrompt)
	_ = command("tmux", "select-window", "-t", session+":"+judgeWindow).Run()
	if !attachEnabled(cfg.file) || cfg.file.Tmux.AttachMode == "none" {
		fmt.Printf("Created detached tmux session: %s\n", session)
		return nil
	}
	if cfg.file.Tmux.AttachMode == "iterm-control-mode" {
		return command("tmux", "-CC", "attach", "-t", session).Run()
	}
	return command("tmux", "attach", "-t", session).Run()
}

func buildJudgePrompt(prompt string, candidates []candidate) string {
	if prompt == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString("You are the judge/coordinator for a coding-agent A/B test.\n\n")
	b.WriteString("Original task:\n")
	b.WriteString(prompt)
	b.WriteString("\n\nContestants:\n")
	for _, cand := range candidates {
		b.WriteString(fmt.Sprintf("- %s in %s (branch %s)\n", cand.agent, cand.path, cand.branch))
	}
	b.WriteString("\nDo not judge yet. Wait until I explicitly tell you the contestants are done and ask you to judge.\n\n")
	b.WriteString("When I ask you to judge: inspect the candidate worktrees, compare their diffs and checks, pick the best one first, then ask before applying it to this base worktree. NEVER delete or clean up any worktree or branch unless I explicitly approve cleanup after your verdict.")
	return b.String()
}

func commandLine(def AgentDef) string {
	parts := []string{shellQuote(def.Command)}
	for _, arg := range def.Args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func sendPrompt(target, prompt string) {
	if target == "" || prompt == "" {
		return
	}
	cmd := exec.Command("sh", "-c", "sleep 2; tmux send-keys -t \"$1\" -l \"$2\"; tmux send-keys -t \"$1\" Enter", "agenttab-send", target, prompt)
	_ = cmd.Start()
}
