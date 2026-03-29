package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"os/exec"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

	dashboard := "none"

	if cfg.Store.Driver == "postgres" {
		fmt.Println("Web dashboards currently only support SQLite. Skipping dashboard selection.")
	} else {
		fmt.Println("Web dashboard (optional, requires Docker):")
		fmt.Println("  1. none         — skip, use TUI only")
		fmt.Println("  2. datasette    — lightweight, SQLite-native, YAML-configured charts")
		fmt.Println("  3. lite-queen   — minimal SQLite browser")

		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("Choose [1-3] (default: 1): ")
			choice, _ := reader.ReadString('\n')
			choice = strings.TrimSpace(choice)

			if choice == "" || choice == "1" {
				dashboard = "none"
				break
			} else if choice == "2" {
				dashboard = "datasette"
				break
			} else if choice == "3" {
				dashboard = "lite-queen"
				break
			} else {
				fmt.Println("invalid choice, please select 1-3")
			}
		}
	}

	if dashboard == "none" {
		fmt.Println("\nselected dashboard: none (TUI only)")
	} else {
		fmt.Printf("\nselected dashboard: %s\n", dashboard)
	}

	if err := updateDashboardConfig(cfgPath, dashboard); err != nil {
		fmt.Printf("warning: failed to update config with dashboard choice: %v\n", err)
	}
	cfg.Retro.Dashboard = dashboard

	if dashboard != "none" {
		dir, _ := kirkupDir()
		composePath := filepath.Join(dir, "dashboard", "docker-compose.yaml")

		// Tear down existing before overwriting if it exists
		if _, err := os.Stat(composePath); err == nil {
			cmd := exec.Command("docker", "compose", "-f", composePath, "down")
			_ = cmd.Run()
		}

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
  port: 8001
`

func updateDashboardConfig(path string, dashboard string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	if len(root.Content) == 0 {
		return nil
	}
	doc := root.Content[0]

	var retroNode *yaml.Node
	for i := 0; i < len(doc.Content)-1; i += 2 {
		key := doc.Content[i]
		if key.Value == "retro" {
			retroNode = doc.Content[i+1]
			break
		}
	}

	if retroNode == nil {
		// Create retro block
		doc.Content = append(doc.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "retro"},
			&yaml.Node{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "dashboard"},
					{Kind: yaml.ScalarNode, Value: dashboard},
				},
			},
		)
	} else {
		// Update existing retro block
		found := false
		for i := 0; i < len(retroNode.Content)-1; i += 2 {
			if retroNode.Content[i].Value == "dashboard" {
				retroNode.Content[i+1].Value = dashboard
				found = true
				break
			}
		}
		if !found {
			retroNode.Content = append(retroNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "dashboard"},
				&yaml.Node{Kind: yaml.ScalarNode, Value: dashboard},
			)
		}
	}

	out, err := yaml.Marshal(&root)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}
