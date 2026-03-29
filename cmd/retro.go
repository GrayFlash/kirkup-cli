package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/config"
	"github.com/GrayFlash/kirkup-cli/internal/retro"
)

var (
	retroWeek    bool
	retroMonth   bool
	retroFrom    string
	retroTo      string
	retroProject string
	retroWeb     bool
	retroWebStop bool
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
	retroCmd.Flags().BoolVar(&retroWeb, "web", false, "Launch the web dashboard via Docker")
	retroCmd.Flags().BoolVar(&retroWebStop, "web-stop", false, "Stop the web dashboard")
	rootCmd.AddCommand(retroCmd)
}

func runRetro(_ *cobra.Command, _ []string) error {
	if retroWebStop {
		return stopWebDashboard()
	}

	if retroWeb {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if cfg.Retro.Dashboard == "none" || cfg.Retro.Dashboard == "" {
			return fmt.Errorf("no web dashboard configured; run 'kirkup init' to select one")
		}
		return launchWebDashboard(cfg)
	}

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

func launchWebDashboard(cfg *config.Config) error {
	dir, err := kirkupDir()
	if err != nil {
		return err
	}
	composePath := filepath.Join(dir, "dashboard", "docker-compose.yaml")

	if err := generateDashboardCompose(cfg, composePath); err != nil {
		return fmt.Errorf("generate compose: %w", err)
	}

	fmt.Println("launching dashboard via docker compose...")
	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start dashboard: %w (is Docker installed and running?)", err)
	}

	fmt.Println("\ndashboard is running!")

	port := cfg.Retro.Port
	if port == 0 {
		port = 8001
	}
	fmt.Printf("access it at: http://localhost:%d\n", port)
	return nil
}

func stopWebDashboard() error {
	dir, err := kirkupDir()
	if err != nil {
		return err
	}
	composePath := filepath.Join(dir, "dashboard", "docker-compose.yaml")

	if _, err := os.Stat(composePath); err != nil {
		fmt.Println("dashboard is not running or not configured")
		return nil
	}

	checkCmd := exec.Command("docker", "compose", "-f", composePath, "ps", "-q")
	if out, err := checkCmd.Output(); err == nil && len(strings.TrimSpace(string(out))) == 0 {
		fmt.Println("dashboard is not running")
		return nil
	}

	fmt.Println("stopping dashboard...")
	cmd := exec.Command("docker", "compose", "-f", composePath, "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop dashboard: %w", err)
	}

	fmt.Println("dashboard stopped.")
	return nil
}
