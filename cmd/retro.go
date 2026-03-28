package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/retro"
)

var (
	retroWeek    bool
	retroMonth   bool
	retroFrom    string
	retroTo      string
	retroProject string
)

var retroCmd = &cobra.Command{
	Use:   "retro",
	Short: "Show a retrospective summary of your engineering activity",
	RunE:  runRetro,
}

func init() {
	retroCmd.Flags().BoolVar(&retroWeek, "week", false, "Current week (default)")
	retroCmd.Flags().BoolVar(&retroMonth, "month", false, "Current month")
	retroCmd.Flags().StringVar(&retroFrom, "from", "", "Start date (YYYY-MM-DD)")
	retroCmd.Flags().StringVar(&retroTo, "to", "", "End date (YYYY-MM-DD)")
	retroCmd.Flags().StringVar(&retroProject, "project", "", "Filter by project")
	rootCmd.AddCommand(retroCmd)
}

func runRetro(_ *cobra.Command, _ []string) error {
	cfg, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	from, to, err := resolveRange()
	if err != nil {
		return err
	}

	summary, err := retro.Aggregate(
		context.Background(), s,
		from, to, retroProject,
		cfg.Sessions.GapThresholdMinutes,
	)
	if err != nil {
		return fmt.Errorf("aggregate: %w", err)
	}

	return retro.Render(os.Stdout, summary)
}

// resolveRange returns the from/to time range based on flags.
// Priority: --from/--to > --month > --week (default).
func resolveRange() (time.Time, time.Time, error) {
	if retroFrom != "" || retroTo != "" {
		return parseDateRange(retroFrom, retroTo)
	}
	if retroMonth {
		return currentMonth()
	}
	return currentWeek()
}

func currentWeek() (time.Time, time.Time, error) {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7 in ISO
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	from := truncateDay(monday)
	to := from.AddDate(0, 0, 6).Add(24*time.Hour - time.Second)
	return from, to, nil
}

func currentMonth() (time.Time, time.Time, error) {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 1, 0).Add(-time.Second)
	return from, to, nil
}


func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
