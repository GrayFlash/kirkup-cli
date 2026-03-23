package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/store"
)

var (
	exportFrom    string
	exportTo      string
	exportProject string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export collected events as JSON",
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFrom, "from", "", "Start date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportTo, "to", "", "End date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportProject, "project", "", "Filter by project")
	rootCmd.AddCommand(exportCmd)
}

func runExport(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = s.Close() }()

	f := store.EventFilter{Project: exportProject}

	if exportFrom != "" {
		t, err := time.ParseInLocation("2006-01-02", exportFrom, time.Local)
		if err != nil {
			return fmt.Errorf("invalid --from date %q: use YYYY-MM-DD", exportFrom)
		}
		f.Since = &t
	}
	if exportTo != "" {
		t, err := time.ParseInLocation("2006-01-02", exportTo, time.Local)
		if err != nil {
			return fmt.Errorf("invalid --to date %q: use YYYY-MM-DD", exportTo)
		}
		end := t.Add(24*time.Hour - time.Second)
		f.Until = &end
	}

	events, err := s.QueryPromptEvents(context.Background(), f)
	if err != nil {
		return fmt.Errorf("query events: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(events)
}
