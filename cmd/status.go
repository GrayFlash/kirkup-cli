package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/agent"
	agentclaude "github.com/GrayFlash/kirkup-cli/agent/claude"
	agentcursor "github.com/GrayFlash/kirkup-cli/agent/cursor"
	agentgemini "github.com/GrayFlash/kirkup-cli/agent/gemini"
	"github.com/GrayFlash/kirkup-cli/store"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status, detected agents, and today's event count",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(_ *cobra.Command, _ []string) error {
	// Daemon status
	pidPath, _ := pidFilePath()
	if pid, err := readPID(pidPath); err == nil && isRunning(pid) {
		fmt.Printf("daemon:  running (pid %d)\n", pid)
	} else {
		fmt.Println("daemon:  stopped")
	}

	// Agent detection
	registry := agent.NewRegistry(
		agentgemini.New(),
		agentcursor.New(),
		agentclaude.New(),
	)
	fmt.Println()
	fmt.Println("agents:")
	for _, a := range registry.All() {
		status := "not detected"
		if a.Detect() {
			status = "detected"
		}
		fmt.Printf("  %-14s %s\n", a.Name(), status)
	}

	// Events today (best-effort — skip if config or store unavailable)
	cfg, err := loadConfig()
	if err != nil {
		return nil
	}
	s, err := openStore(cfg)
	if err != nil {
		return nil
	}
	defer func() { _ = s.Close() }()

	midnight := today()
	events, err := s.QueryPromptEvents(context.Background(), store.EventFilter{Since: &midnight})
	if err != nil {
		return nil
	}
	fmt.Printf("\nevents today: %d\n", len(events))
	return nil
}

// today returns midnight of the current local day in UTC.
func today() time.Time {
	now := time.Now()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}
