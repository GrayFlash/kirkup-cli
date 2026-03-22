package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/agent"
	agentcursor "github.com/GrayFlash/kirkup-cli/agent/cursor"
	agentgemini "github.com/GrayFlash/kirkup-cli/agent/gemini"
	"github.com/GrayFlash/kirkup-cli/store/sqlite"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create config file, initialise database, and detect agents",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(_ *cobra.Command, _ []string) error {
	cfgPath, err := defaultConfigPath()
	if err != nil {
		return err
	}

	// -- Config file --
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("config already exists: %s\n", cfgPath)
	} else {
		if err := writeDefaultConfig(cfgPath); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Printf("created config:        %s\n", cfgPath)
	}

	// -- Database --
	dbPath, err := defaultDBPath()
	if err != nil {
		return err
	}
	s, err := sqlite.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = s.Close() }()

	if err := s.Migrate(context.Background()); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	fmt.Printf("initialised database:  %s\n", dbPath)

	// -- Agent detection --
	registry := agent.NewRegistry(
		agentgemini.New(),
		agentcursor.New(),
	)

	fmt.Println()
	fmt.Println("agents:")
	for _, a := range registry.All() {
		status := "not detected"
		if a.Detect() {
			status = "detected ✓"
		}
		fmt.Printf("  %-14s %s\n", a.Name(), status)
	}

	fmt.Println()
	fmt.Println("run \"kirkup start\" to begin collecting.")
	return nil
}

// defaultConfigPath returns ~/.kirkup/config.yaml.
func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kirkup", "config.yaml"), nil
}

// defaultDBPath returns ~/.kirkup/kirkup.db.
func defaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kirkup", "kirkup.db"), nil
}

// writeDefaultConfig copies the embedded default config to dst.
func writeDefaultConfig(dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	data, err := defaultConfigBytes()
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// defaultConfigBytes reads configs/default.yaml relative to the binary.
// Falls back to an inline minimal config if the file is not found.
func defaultConfigBytes() ([]byte, error) {
	// Try reading from the embedded path first (works when running from repo root).
	candidates := []string{
		"configs/default.yaml",
		filepath.Join(filepath.Dir(os.Args[0]), "configs", "default.yaml"),
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err == nil {
			return data, nil
		}
	}
	// Inline fallback so `kirkup init` always works regardless of install method.
	return []byte(minimalConfig), nil
}

const minimalConfig = `# ~/.kirkup/config.yaml
store:
  driver: sqlite
  sqlite:
    path: ~/.kirkup/kirkup.db
agents:
  gemini-cli:
    enabled: true
  cursor:
    enabled: true
classifier:
  mode: rules
sessions:
  gap_threshold_minutes: 30
daemon:
  poll_interval_seconds: 5
  log_level: info
`
