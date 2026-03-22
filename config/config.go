package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Store    StoreConfig            `yaml:"store"`
	Agents   map[string]AgentConfig `yaml:"agents"`
	Daemon   DaemonConfig           `yaml:"daemon"`
	Projects []ProjectConfig        `yaml:"projects"`
	Sessions SessionsConfig         `yaml:"sessions"`
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
	Driver string       `yaml:"driver"`
	SQLite SQLiteConfig `yaml:"sqlite"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type AgentConfig struct {
	Enabled  bool     `yaml:"enabled"`
	LogPaths []string `yaml:"log_paths"`
}

type DaemonConfig struct {
	PollIntervalSeconds int    `yaml:"poll_interval_seconds"`
	LogLevel            string `yaml:"log_level"`
}

func Load(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(expandHome(path))
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.Store.SQLite.Path = expandHome(cfg.Store.SQLite.Path)
	for name, agent := range cfg.Agents {
		for i, p := range agent.LogPaths {
			agent.LogPaths[i] = expandHome(p)
		}
		cfg.Agents[name] = agent
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

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, path[2:])
}
