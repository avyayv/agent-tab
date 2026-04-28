package agenttab

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func requireCommands(cmds []string) error {
	seen := map[string]bool{}
	for _, name := range cmds {
		if seen[name] || name == "" {
			continue
		}
		seen[name] = true
		if _, err := exec.LookPath(name); err != nil {
			return fmt.Errorf("missing command: %s", name)
		}
	}
	return nil
}

func command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd
}

func commandIn(dir, name string, args ...string) *exec.Cmd {
	cmd := command(name, args...)
	cmd.Dir = dir
	return cmd
}

func output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, stderr.String())
	}
	return string(out), nil
}

func outputIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, stderr.String())
	}
	return string(out), nil
}

func shellCmd(fc FileConfig, cmd string) string {
	return fc.Shell + " -lc " + shellQuote(cmd+"; exec "+fc.Shell+" -l")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func findExecutable(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return name
	}
	return path
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return h
}

func defaultConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "agent-tab", "config.yaml")
	}
	return filepath.Join(homeDir(), ".config", "agent-tab", "config.yaml")
}

func expandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" {
		return homeDir(), nil
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir(), strings.TrimPrefix(path, "~/")), nil
	}
	return filepath.Abs(path)
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}
