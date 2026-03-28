package cmd

import (
	"context"
	"encoding/csv"
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
	exportFormat  string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export prompt events to JSON or CSV",
	RunE:  runExport,
}

func init() {
	exportCmd.Flags().StringVar(&exportFrom, "from", "", "Start date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportTo, "to", "", "End date (YYYY-MM-DD)")
	exportCmd.Flags().StringVar(&exportProject, "project", "", "Filter by project")
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Output format: json or csv")
	rootCmd.AddCommand(exportCmd)
}

func runExport(_ *cobra.Command, _ []string) error {
	_, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	f := store.EventFilter{Project: exportProject}

	from, to, err := parseDateRange(exportFrom, exportTo)
	if err != nil {
		return err
	}
	if !from.IsZero() {
		f.Since = &from
	}
	if !to.IsZero() {
		f.Until = &to
	}

	events, err := s.QueryPromptEvents(context.Background(), f)
	if err != nil {
		return fmt.Errorf("query events: %w", err)
	}

	switch exportFormat {
	case "csv":
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{"id", "timestamp", "agent", "session_id", "project", "git_branch", "git_remote", "working_dir", "prompt"})
		for _, e := range events {
			_ = w.Write([]string{
				e.ID,
				e.Timestamp.UTC().Format(time.RFC3339),
				e.Agent,
				e.SessionID,
				e.Project,
				e.GitBranch,
				e.GitRemote,
				e.WorkingDir,
				e.Prompt,
			})
		}
		w.Flush()
		return w.Error()
	case "json", "":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(events)
	default:
		return fmt.Errorf("unknown format %q: use json or csv", exportFormat)
	}
}
