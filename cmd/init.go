package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	fmt.Println("Web dashboard (optional, requires Docker):")
	fmt.Println("  1. none         — skip, use TUI only")
	fmt.Println("  2. datasette    — lightweight, SQLite-native, YAML-configured charts")
	fmt.Println("  3. lite-queen   — minimal SQLite browser")
	fmt.Print("Choose [1-3] (default: 1): ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	dashboard := "none"
	switch choice {
	case "2":
		dashboard = "datasette"
	case "3":
		dashboard = "lite-queen"
	}

	if err := updateDashboardConfig(cfgPath, dashboard); err != nil {
		fmt.Printf("warning: failed to update config with dashboard choice: %v\n", err)
	}
	cfg.Retro.Dashboard = dashboard

	if dashboard != "none" {
		dir, _ := kirkupDir()
		composePath := filepath.Join(dir, "dashboard", "docker-compose.yaml")
		if err := generateDashboardCompose(cfg, composePath); err != nil {
			fmt.Printf("warning: failed to generate docker-compose.yaml: %v\n", err)
		} else {
			fmt.Printf("generated compose file at %s\n", composePath)
		}
	}

	fmt.Println()
	fmt.Println("run \"kirkup start\" to begin collecting.")
	return nil
}

// defaultConfigPath returns ~/.kirkup/config.yaml.

// defaultDBPath returns ~/.kirkup/kirkup.db.

// writeDefaultConfig writes data to dst, creating parent dirs as needed.
func writeDefaultConfig(dst string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
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
retro:
  dashboard: none
`

func updateDashboardConfig(path string, dashboard string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)

	if !strings.Contains(content, "retro:") {
		content += "\nretro:\n  dashboard: " + dashboard + "\n"
	} else {
		lines := strings.Split(content, "\n")
		inRetro := false
		found := false
		for i, line := range lines {
			if strings.HasPrefix(line, "retro:") {
				inRetro = true
				continue
			}
			if inRetro {
				if strings.HasPrefix(line, "  dashboard:") {
					lines[i] = "  dashboard: " + dashboard
					found = true
					break
				}
				if !strings.HasPrefix(line, "  ") && line != "" {
					break
				}
			}
		}
		if found {
			content = strings.Join(lines, "\n")
		} else {
			content += "\nretro:\n  dashboard: " + dashboard + "\n"
		}
	}

	return os.WriteFile(path, []byte(content), 0o600)
}
