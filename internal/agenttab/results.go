package agenttab

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type resultsDB struct {
	Version int         `json:"version"`
	Runs    []resultRun `json:"runs"`
}

type resultRun struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Repo      string    `json:"repo,omitempty"`
	Branch    string    `json:"branch,omitempty"`
	TaskType  string    `json:"task_type"`
	Order     []string  `json:"order"`
	Notes     string    `json:"notes,omitempty"`
}

type recordOptions struct {
	configPath  string
	resultsFile string
	taskType    string
	order       []string
	notes       string
}

type statsOptions struct {
	configPath  string
	resultsFile string
	taskType    string
}

func recordCommand(args []string) error {
	opts, err := parseRecordOptions(args)
	if err != nil {
		return err
	}
	fc, err := configForResults(opts.configPath, opts.resultsFile)
	if err != nil {
		return err
	}
	path, err := expandPath(fc.ResultsFile)
	if err != nil {
		return err
	}
	db, err := loadResults(path)
	if err != nil {
		return err
	}
	repo, _ := output("git", "rev-parse", "--show-toplevel")
	branch, _ := output("git", "rev-parse", "--abbrev-ref", "HEAD")
	run := resultRun{
		ID:        fmt.Sprintf("%s-%d", time.Now().Format("20060102-150405"), os.Getpid()),
		Timestamp: time.Now().UTC(),
		Repo:      strings.TrimSpace(repo),
		Branch:    strings.TrimSpace(branch),
		TaskType:  opts.taskType,
		Order:     opts.order,
		Notes:     opts.notes,
	}
	db.Version = 1
	db.Runs = append(db.Runs, run)
	if err := saveResults(path, db); err != nil {
		return err
	}
	fmt.Printf("Recorded result in %s\n", path)
	fmt.Printf("%s: %s\n", run.TaskType, strings.Join(run.Order, " > "))
	return nil
}

func statsCommand(args []string) error {
	opts, err := parseStatsOptions(args)
	if err != nil {
		return err
	}
	fc, err := configForResults(opts.configPath, opts.resultsFile)
	if err != nil {
		return err
	}
	path, err := expandPath(fc.ResultsFile)
	if err != nil {
		return err
	}
	db, err := loadResults(path)
	if err != nil {
		return err
	}
	if len(db.Runs) == 0 {
		fmt.Printf("No results recorded yet (%s)\n", path)
		return nil
	}
	type score struct{ Wins, Seconds, Thirds, Points, Runs int }
	scores := map[string]*score{}
	total := 0
	for _, run := range db.Runs {
		if opts.taskType != "" && run.TaskType != opts.taskType {
			continue
		}
		total++
		for i, agent := range run.Order {
			if scores[agent] == nil {
				scores[agent] = &score{}
			}
			s := scores[agent]
			s.Runs++
			s.Points += len(run.Order) - i
			switch i {
			case 0:
				s.Wins++
			case 1:
				s.Seconds++
			case 2:
				s.Thirds++
			}
		}
	}
	if total == 0 {
		fmt.Printf("No results for task type %q (%s)\n", opts.taskType, path)
		return nil
	}
	agents := make([]string, 0, len(scores))
	for agent := range scores {
		agents = append(agents, agent)
	}
	sort.Slice(agents, func(i, j int) bool {
		a, b := scores[agents[i]], scores[agents[j]]
		if a.Wins != b.Wins {
			return a.Wins > b.Wins
		}
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		return agents[i] < agents[j]
	})
	label := "all task types"
	if opts.taskType != "" {
		label = opts.taskType
	}
	fmt.Printf("Results for %s (%d runs)\n", label, total)
	fmt.Println("agent\twins\tseconds\tthirds\tpoints\truns")
	for _, agent := range agents {
		s := scores[agent]
		fmt.Printf("%s\t%d\t%d\t%d\t%d\t%d\n", agent, s.Wins, s.Seconds, s.Thirds, s.Points, s.Runs)
	}
	return nil
}

func parseRecordOptions(args []string) (recordOptions, error) {
	opts := recordOptions{taskType: "unspecified"}
	for i := 0; i < len(args); i++ {
		name, value, ok := strings.Cut(strings.TrimPrefix(args[i], "--"), "=")
		if !strings.HasPrefix(args[i], "--") {
			return opts, fmt.Errorf("unexpected argument: %s", args[i])
		}
		take := func() (string, error) {
			if ok {
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
			v, err := take()
			if err != nil {
				return opts, err
			}
			opts.configPath = v
		case "results-file":
			v, err := take()
			if err != nil {
				return opts, err
			}
			opts.resultsFile = v
		case "task-type", "type":
			v, err := take()
			if err != nil {
				return opts, err
			}
			opts.taskType = v
		case "order":
			v, err := take()
			if err != nil {
				return opts, err
			}
			opts.order = splitCSV(v)
		case "notes", "note":
			v, err := take()
			if err != nil {
				return opts, err
			}
			opts.notes = v
		case "help", "h":
			recordUsage()
			os.Exit(0)
		default:
			return opts, fmt.Errorf("unknown option: --%s", name)
		}
	}
	if len(opts.order) < 2 {
		return opts, errors.New("record requires --order winner,second[,third]")
	}
	return opts, nil
}

func parseStatsOptions(args []string) (statsOptions, error) {
	opts := statsOptions{}
	for i := 0; i < len(args); i++ {
		name, value, ok := strings.Cut(strings.TrimPrefix(args[i], "--"), "=")
		if !strings.HasPrefix(args[i], "--") {
			return opts, fmt.Errorf("unexpected argument: %s", args[i])
		}
		take := func() (string, error) {
			if ok {
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
			v, err := take()
			if err != nil {
				return opts, err
			}
			opts.configPath = v
		case "results-file":
			v, err := take()
			if err != nil {
				return opts, err
			}
			opts.resultsFile = v
		case "task-type", "type":
			v, err := take()
			if err != nil {
				return opts, err
			}
			opts.taskType = v
		case "help", "h":
			statsUsage()
			os.Exit(0)
		default:
			return opts, fmt.Errorf("unknown option: --%s", name)
		}
	}
	return opts, nil
}

func recordUsage() {
	fmt.Println("Usage: agent-tab record --task-type TYPE --order winner,second[,third] [--notes TEXT]")
}
func statsUsage() { fmt.Println("Usage: agent-tab stats [--task-type TYPE]") }

func configForResults(configPath, resultsFile string) (FileConfig, error) {
	fc := defaultConfig()
	if configPath == "" {
		configPath = os.Getenv("AGENT_TAB_CONFIG")
	}
	if err := loadConfigFile(&fc, configPath); err != nil {
		return fc, err
	}
	applyEnv(&fc)
	if resultsFile != "" {
		fc.ResultsFile = resultsFile
	}
	return fc, nil
}

func loadResults(path string) (resultsDB, error) {
	var db resultsDB
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return resultsDB{Version: 1}, nil
		}
		return db, err
	}
	if len(data) == 0 {
		return resultsDB{Version: 1}, nil
	}
	if err := json.Unmarshal(data, &db); err != nil {
		return db, err
	}
	if db.Version == 0 {
		db.Version = 1
	}
	return db, nil
}

func saveResults(path string, db resultsDB) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := []string{}
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}
