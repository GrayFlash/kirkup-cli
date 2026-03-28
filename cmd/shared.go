package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/GrayFlash/kirkup-cli/agent"
	"github.com/GrayFlash/kirkup-cli/agent/claude"
	"github.com/GrayFlash/kirkup-cli/agent/cursor"
	"github.com/GrayFlash/kirkup-cli/agent/gemini"
	"github.com/GrayFlash/kirkup-cli/agent/generic"
	"github.com/GrayFlash/kirkup-cli/config"
	"github.com/GrayFlash/kirkup-cli/store"
	"github.com/GrayFlash/kirkup-cli/store/postgres"
	"github.com/GrayFlash/kirkup-cli/store/sqlite"
)

func newAgentRegistry(cfg *config.Config) *agent.Registry {
	var adapters []agent.Adapter

	// Built-in agents
	builtins := map[string]func() agent.Adapter{
		"gemini-cli":  func() agent.Adapter { return gemini.New() },
		"cursor":      func() agent.Adapter { return cursor.New() },
		"claude-code": func() agent.Adapter { return claude.New() },
	}

	for name, factory := range builtins {
		enabled := true
		if cfg != nil {
			if ac, ok := cfg.Agents[name]; ok {
				enabled = ac.Enabled
			}
		}
		if enabled {
			adapters = append(adapters, factory())
		}
	}

	if cfg != nil {
		for name, ac := range cfg.Agents {
			if _, isBuiltin := builtins[name]; isBuiltin {
				continue
			}
			if ac.Enabled && ac.PromptField != "" {
				adapters = append(adapters, generic.New(name, ac))
			}
		}
	}

	return agent.NewRegistry(adapters...)
}

func openApp() (*config.Config, store.Store, func(), error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, nil, nil, err
	}
	s, err := openStore(cfg)
	if err != nil {
		return nil, nil, nil, err
	}
	cleanup := func() { _ = s.Close() }
	return cfg, s, cleanup, nil
}

func openInEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		return fmt.Errorf("$EDITOR is not set; open %s manually", path)
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func parseDateRange(fromStr, toStr string) (time.Time, time.Time, error) {
	var from, to time.Time
	var err error
	if fromStr != "" {
		from, err = time.ParseInLocation("2006-01-02", fromStr, time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from date: %w", err)
		}
	}
	if toStr != "" {
		to, err = time.ParseInLocation("2006-01-02", toStr, time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to date: %w", err)
		}
		// Include the entire 'to' day
		to = to.Add(24*time.Hour - time.Nanosecond)
	} else if fromStr != "" {
		// If only 'from' is provided, default 'to' to today
		to = time.Now().Local()
	}
	return from, to, nil
}

func kirkupDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}
	return filepath.Join(home, ".kirkup"), nil
}

func defaultConfigPath() (string, error) {
	dir, err := kirkupDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func defaultDBPath() (string, error) {
	dir, err := kirkupDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kirkup.db"), nil
}

func pidFilePath() (string, error) {
	dir, err := kirkupDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kirkup.pid"), nil
}

func logFilePath() (string, error) {
	dir, err := kirkupDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kirkup.log"), nil
}

func loadConfig() (*config.Config, error) {
	path, err := defaultConfigPath()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		if os.IsNotExist(err) {
			tmp := &config.Config{}
			tmp.Daemon.PollIntervalSeconds = 5
			tmp.Daemon.LogLevel = "info"
			tmp.Sessions.GapThresholdMinutes = 30
			tmp.Store.Driver = "sqlite"
			dbPath, _ := defaultDBPath()
			tmp.Store.SQLite.Path = dbPath
			return tmp, nil
		}
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

func openStore(cfg *config.Config) (store.Store, error) {
	switch cfg.Store.Driver {
	case "postgres":
		return postgres.Open(cfg.Store.PG.DSN)
	case "sqlite", "":
		s, err := sqlite.Open(cfg.Store.SQLite.Path)
		if err != nil {
			return nil, err
		}
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported store driver: %q", cfg.Store.Driver)
	}
}

func today() time.Time {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) > max && max > 0 {
		return string(runes[:max-1]) + "…"
	} else if len(runes) > max && max <= 0 {
		return ""
	}
	return s
}
