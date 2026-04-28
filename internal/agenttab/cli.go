package agenttab

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func parseCLI(args []string) (cliOptions, error) {
	opts := cliOptions{attach: true}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			opts.prompt = strings.Join(args[i+1:], " ")
			break
		}
		if arg == "--help" || arg == "-h" {
			usage()
			os.Exit(0)
		}
		if !strings.HasPrefix(arg, "--") {
			opts.positionals = append(opts.positionals, arg)
			continue
		}
		name, value, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		takeValue := func() (string, error) {
			if hasValue {
				return value, nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("--%s requires a value", name)
			}
			i++
			return args[i], nil
		}
		switch name {
		case "config":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.configPath = v
		case "worktrees-dir":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.worktreesDir = v
		case "judge":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.judge = v
		case "session":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.session = v
		case "layout":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.layout = v
		case "attach-mode":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.attachMode = v
		case "agents":
			v, err := takeValue()
			if err != nil {
				return opts, err
			}
			opts.agentsFlag = v
		case "attach":
			opts.attachSet = true
			opts.attach = true
		case "no-attach":
			opts.attachSet = true
			opts.attach = false
		case "dry-run":
			opts.dryRun = true
		case "show-config":
			opts.showConfig = true
		default:
			return opts, fmt.Errorf("unknown option: --%s", name)
		}
	}
	return opts, nil
}

func buildRunConfig(fc FileConfig, opts cliOptions) (config, error) {
	cfg := config{file: fc, prompt: opts.prompt, session: opts.session, dryRun: opts.dryRun}
	if cfg.session == "" {
		cfg.session = sessionFromPositionals(opts.positionals)
	}
	if opts.agentsFlag != "" {
		for _, a := range strings.Split(opts.agentsFlag, ",") {
			if strings.TrimSpace(a) != "" {
				cfg.agents = append(cfg.agents, strings.TrimSpace(a))
			}
		}
	} else if len(opts.positionals) > 0 && opts.positionals[0] == "all" {
		cfg.agents = configuredAgentNames(fc)
		if len(opts.positionals) > 2 {
			return cfg, errors.New("usage: agenttab all [session_name] [-- prompt]")
		}
		if len(opts.positionals) == 2 {
			cfg.session = opts.positionals[1]
		}
	} else {
		for _, arg := range opts.positionals {
			if _, ok := fc.Agents[arg]; ok {
				if cfg.session != "" && cfg.session != opts.session {
					return cfg, errors.New("agent names must come before session_name")
				}
				cfg.agents = append(cfg.agents, arg)
			} else if cfg.session == "" {
				cfg.session = arg
			} else if cfg.session != arg {
				return cfg, fmt.Errorf("unexpected argument: %s", arg)
			}
		}
	}
	if len(cfg.agents) == 0 {
		cfg.agents = []string{"codex", "pi"}
	}
	if len(cfg.agents) < 2 || len(cfg.agents) > 3 {
		return cfg, errors.New("pick two or three agents, or use: agenttab all")
	}
	seen := map[string]bool{}
	for _, agent := range cfg.agents {
		if seen[agent] {
			return cfg, errors.New("pick different agents")
		}
		seen[agent] = true
		def, ok := fc.Agents[agent]
		if !ok || def.Command == "" {
			return cfg, fmt.Errorf("agent %q is not configured", agent)
		}
	}
	if _, ok := fc.Agents[fc.Judge.Agent]; !ok {
		return cfg, fmt.Errorf("judge agent %q is not configured", fc.Judge.Agent)
	}
	if fc.Tmux.Layout == "" {
		return cfg, errors.New("tmux.layout cannot be empty")
	}
	if fc.Tmux.AttachMode != "normal" && fc.Tmux.AttachMode != "iterm-control-mode" && fc.Tmux.AttachMode != "none" {
		return cfg, errors.New("tmux.attach_mode must be normal, iterm-control-mode, or none")
	}
	return cfg, nil
}

func sessionFromPositionals(pos []string) string { return "" }

func usage() {
	fmt.Println("Usage: agenttab [flags] [all|agent...] [session_name] [-- prompt]")
	fmt.Println("Flags:")
	fmt.Println("  --config PATH")
	fmt.Println("  --worktrees-dir PATH")
	fmt.Println("  --judge AGENT")
	fmt.Println("  --session NAME")
	fmt.Println("  --agents a,b[,c]")
	fmt.Println("  --layout tiled|even-horizontal|even-vertical")
	fmt.Println("  --attach-mode normal|iterm-control-mode|none")
	fmt.Println("  --attach / --no-attach")
	fmt.Println("  --dry-run")
	fmt.Println("  --show-config")
	fmt.Println("Examples:")
	fmt.Println("  agenttab")
	fmt.Println("  agenttab codex claude -- 'implement X'")
	fmt.Println("  agenttab all -- 'implement X'")
}
