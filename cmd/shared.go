package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GrayFlash/kirkup-cli/config"
	"github.com/GrayFlash/kirkup-cli/store"
	"github.com/GrayFlash/kirkup-cli/store/sqlite"
)

func loadConfig() (*config.Config, error) {
	cfgPath, err := defaultConfigPath()
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("%w\nrun \"kirkup init\" to create it", err)
	}
	return cfg, nil
}

func openStore(cfg *config.Config) (store.Store, error) {
	switch cfg.Store.Driver {
	case "sqlite", "":
		return sqlite.Open(cfg.Store.SQLite.Path)
	default:
		return nil, fmt.Errorf("unsupported store driver: %s", cfg.Store.Driver)
	}
}

func pidFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kirkup", "kirkup.pid"), nil
}

func logFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kirkup", "kirkup.log"), nil
}

func writePID(path string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0o644)
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
