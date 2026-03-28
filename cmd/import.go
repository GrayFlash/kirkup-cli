package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/internal/collector"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import historical events from agent log files",
	Long: `Import historical events from all detected agent log files.
This command scans all matching log files, deduplicates against existing
events in the database, and stores any new events found.`,
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(_ *cobra.Command, _ []string) error {
	cfg, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()

	registry := newAgentRegistry(cfg)
	c := collector.New(registry, s, cfg, nil)

	fmt.Println("loading existing events for deduplication...")
	if err := c.LoadSeen(ctx); err != nil {
		return err
	}

	fmt.Println("scanning agent logs...")
	total, newCount := c.Scan(ctx)

	fmt.Printf("\nimport complete:\n")
	fmt.Printf("  total events processed: %d\n", total)
	fmt.Printf("  new events imported:    %d\n", newCount)

	if newCount > 0 {
		fmt.Println("\nrun 'kirkup classify' to categorize new events.")
	}

	return nil
}
