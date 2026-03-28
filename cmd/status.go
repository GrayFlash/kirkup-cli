package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

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

	// Events today
	cfg, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	// Agent detection
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

	midnight := today()
	events, err := s.QueryPromptEvents(context.Background(), store.EventFilter{Since: &midnight})
	if err != nil {
		return err
	}
	fmt.Printf("\nevents today: %d\n", len(events))
	return nil
}
