package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/models"
)

var (
	logProject string
	logTime    string
)

var logCmd = &cobra.Command{
	Use:   "log <description>",
	Short: "Manually log engineering activity",
	Long: `Manually log activity that isn't captured by agent logs, 
such as spec reading, planning, or meetings.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runLog,
}

func init() {
	logCmd.Flags().StringVarP(&logProject, "project", "p", "", "Project name")
	logCmd.Flags().StringVarP(&logTime, "time", "t", "", "Time of activity (YYYY-MM-DD HH:MM:SS), defaults to now")
	rootCmd.AddCommand(logCmd)
}

func runLog(_ *cobra.Command, args []string) error {
	_, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	description := args[0]
	ts := time.Now().UTC()

	if logTime != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", logTime, time.Local)
		if err != nil {
			return fmt.Errorf("invalid time format: %w", err)
		}
		ts = t.UTC()
	}

	e := &models.PromptEvent{
		Agent:     "manual",
		Prompt:    description,
		Timestamp: ts,
		Project:   logProject,
	}

	if err := s.InsertPromptEvent(context.Background(), e); err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	fmt.Printf("Logged activity: %s\n", description)
	fmt.Println("Run 'kirkup classify' to categorize this new activity.")

	return nil
}
