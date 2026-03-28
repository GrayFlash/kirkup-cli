package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// DefaultConfig is set by main.go via go:embed so the binary always carries
// the full default config regardless of install location.
var DefaultConfig []byte

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
		if err := writeDefaultConfig(cfgPath, defaultConfigBytes()); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		fmt.Printf("created config:        %s\n", cfgPath)
	}

	// -- Database --
	cfg, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := s.Migrate(context.Background()); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	
	if cfg.Store.Driver == "postgres" {
		fmt.Println("initialised postgres database")
	} else {
		fmt.Printf("initialised database:  %s\n", cfg.Store.SQLite.Path)
	}

	// -- Agent detection --
	registry := newAgentRegistry(cfg)

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

// defaultDBPath returns ~/.kirkup/kirkup.db.

// writeDefaultConfig writes data to dst, creating parent dirs as needed.
func writeDefaultConfig(dst string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func defaultConfigBytes() []byte {
	if len(DefaultConfig) > 0 {
		return DefaultConfig
	}
	// Fallback for go run / tests where embed is not set.
	if data, err := os.ReadFile("config/defaults/default.yaml"); err == nil {
		return data
	}
	return []byte(minimalConfig)
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
