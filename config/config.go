package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Store      StoreConfig            `yaml:"store"`
	Agents     map[string]AgentConfig `yaml:"agents"`
	Daemon     DaemonConfig           `yaml:"daemon"`
	Projects   []ProjectConfig        `yaml:"projects"`
	Sessions   SessionsConfig         `yaml:"sessions"`
	Classifier ClassifierConfig       `yaml:"classifier"`
	Privacy    PrivacyConfig          `yaml:"privacy"`
}

type PrivacyConfig struct {
	Redact   bool     `yaml:"redact"`
	Patterns []string `yaml:"patterns"`
}

type ClassifierConfig struct {
	Mode        string       `yaml:"mode"`
	CustomRules []RuleConfig `yaml:"custom_rules"`
	LLM         LLMConfig    `yaml:"llm"`
}

type RuleConfig struct {
	Category string   `yaml:"category"`
	Keywords []string `yaml:"keywords"`
	Patterns []string `yaml:"patterns"`
	Priority int      `yaml:"priority"`
}

type LLMConfig struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	Endpoint  string `yaml:"endpoint"`
	APIKey    string `yaml:"api_key"`
	BatchSize int    `yaml:"batch_size"`
}

type ProjectConfig struct {
	Name        string       `yaml:"name"`
	DisplayName string       `yaml:"display_name"`
	Match       ProjectMatch `yaml:"match"`
}

type ProjectMatch struct {
	GitRemote string   `yaml:"git_remote"`
	Paths     []string `yaml:"paths"`
}

type SessionsConfig struct {
	GapThresholdMinutes int `yaml:"gap_threshold_minutes"`
}

type StoreConfig struct {
	Driver string         `yaml:"driver"` // sqlite or postgres
	SQLite SQLiteConfig   `yaml:"sqlite"`
	PG     PostgresConfig `yaml:"postgres"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

type AgentConfig struct {
	Enabled  bool     `yaml:"enabled"`
	LogPaths []string `yaml:"log_paths"`

	// Generic parsing support
	Format         string `yaml:"format"`           // "json" or "jsonl"
	PromptField    string `yaml:"prompt_field"`
	TimestampField string `yaml:"timestamp_field"`
	SessionIDField string `yaml:"session_id_field"`
	RoleField      string `yaml:"role_field"`
	UserRoleValue  string `yaml:"user_role_value"`
}

type DaemonConfig struct {
	PollIntervalSeconds int    `yaml:"poll_interval_seconds"`
	LogLevel            string `yaml:"log_level"`
}

func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(ExpandHome(path))
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.Store.SQLite.Path = ExpandHome(cfg.Store.SQLite.Path)
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]AgentConfig)
	}
	for name, agent := range cfg.Agents {
		for i, p := range agent.LogPaths {
			agent.LogPaths[i] = ExpandHome(p)
		}
		cfg.Agents[name] = agent
	}

	if cfg.Daemon.PollIntervalSeconds < 1 {
		cfg.Daemon.PollIntervalSeconds = 1
	}
	if cfg.Sessions.GapThresholdMinutes < 1 {
		cfg.Sessions.GapThresholdMinutes = 1
	}

	// Validate driver and mode
	switch cfg.Store.Driver {
	case "sqlite", "postgres":
	default:
		cfg.Store.Driver = "sqlite"
	}

	switch cfg.Classifier.Mode {
	case "rules", "llm", "both":
	default:
		cfg.Classifier.Mode = "rules"
	}

	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Store: StoreConfig{
			Driver: "sqlite",
			SQLite: SQLiteConfig{
				Path: "~/.kirkup/kirkup.db",
			},
		},
		Daemon: DaemonConfig{
			PollIntervalSeconds: 5,
			LogLevel:            "info",
		},
		Sessions: SessionsConfig{
			GapThresholdMinutes: 30,
		},
	}
}

func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path // Fallback to raw if home can't be resolved
	}
	return filepath.Join(home, path[2:])
}
