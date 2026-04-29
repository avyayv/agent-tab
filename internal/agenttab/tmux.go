package agenttab

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
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
	fmt.Printf("Opened %s contestants. Starting %s judge here.\n", strings.Join(agentLabels(cfg.agents), ", "), cfg.file.Judge.Agent)
	judgeCmd := commandLine(cfg.file.Agents[cfg.file.Judge.Agent])
	return syscall.Exec(findExecutable(cfg.file.Shell), []string{cfg.file.Shell, "-lc", judgeCmd}, os.Environ())
}

func openNewTmuxSession(sourceDir string, cfg config, candidates []candidate, judgePrompt string) error {
	base := cfg.session
	if base == "" {
		base = "agent-tab-ab-test"
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
	var b strings.Builder
	b.WriteString("You are the judge/coordinator for a coding-agent A/B test.\n\n")
	if prompt != "" {
		b.WriteString("Original task:\n")
		b.WriteString(prompt)
		b.WriteString("\n\n")
	}
	b.WriteString("Contestants:\n")
	for _, cand := range candidates {
		b.WriteString(fmt.Sprintf("- %s in %s (branch %s)\n", cand.codename, cand.path, cand.branch))
	}
	b.WriteString("\nDo not judge yet. Wait until I explicitly tell you the contestants are done and ask you to judge.\n\n")
	b.WriteString("When I ask you to judge: inspect the candidate worktrees, compare their diffs and checks, pick the best codename first, then ask before applying it to this base worktree. After you give the final ranking, record the result with: agent-tab record --task-type <type> --order <winner-codename,second-codename,third-codename> --notes '<short reason>'. NEVER delete or clean up any worktree or branch unless I explicitly approve cleanup after your verdict.")
	return b.String()
}

func commandLine(def AgentDef) string {
	return commandLineForSpec(def, agentSpec{})
}

func commandLineForSpec(def AgentDef, spec agentSpec) string {
	parts := []string{shellQuote(def.Command)}
	for _, arg := range def.Args {
		parts = append(parts, shellQuote(arg))
	}
	if spec.Model != "" {
		parts = append(parts, shellQuote(def.ModelArg), shellQuote(spec.Model))
	}
	return strings.Join(parts, " ")
}

func agentLabels(agents []agentSpec) []string {
	labels := make([]string, 0, len(agents))
	for _, agent := range agents {
		labels = append(labels, agent.Label)
	}
	return labels
}

func sendPrompt(target, prompt string) {
	if target == "" || prompt == "" {
		return
	}

	promptFile, err := os.CreateTemp("", "agent-tab-prompt-*.txt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "agent-tab: failed to create prompt file for %s: %v\n", target, err)
		return
	}
	if _, err := promptFile.WriteString(prompt); err != nil {
		fmt.Fprintf(os.Stderr, "agent-tab: failed to write prompt file for %s: %v\n", target, err)
		_ = promptFile.Close()
		_ = os.Remove(promptFile.Name())
		return
	}
	_ = promptFile.Close()

	// This must be an external background process, not a goroutine. In the
	// inside-tmux path agent-tab execs into the judge agent, which kills goroutines
	// before their sleep finishes. Use tmux buffers instead of send-keys -l so
	// quotes/newlines in prompts cannot confuse a shell or tmux argument parser.
	delay := time.Duration(len([]rune(prompt))/200+1) * 500 * time.Millisecond
	if delay > 3*time.Second {
		delay = 3 * time.Second
	}
	bufferName := "agent-tab-" + sanitize(strings.TrimPrefix(target, "%"))
	script := `
		sleep 3
		tmux load-buffer -b "$3" "$2" || exit 1
		tmux paste-buffer -t "$1" -b "$3" || exit 1
		sleep "$4"
		tmux send-keys -t "$1" C-m || exit 1
		tmux delete-buffer -b "$3" >/dev/null 2>&1 || true
		rm -f "$2"
	`
	cmd := exec.Command("sh", "-c", script, "agent-tab-send", target, promptFile.Name(), bufferName, fmt.Sprintf("%.3f", delay.Seconds()))
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "agent-tab: failed to schedule prompt for %s: %v\n", target, err)
		_ = os.Remove(promptFile.Name())
	}
}
