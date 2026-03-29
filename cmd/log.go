package cmd

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/classifier"
	"github.com/GrayFlash/kirkup-cli/models"
)

var (
	logProject  string
	logTime     string
	logCategory string
	logDuration string
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
	logCmd.Flags().StringVarP(&logCategory, "category", "c", "", "Category for this activity (e.g. coding, review, etc)")
	logCmd.Flags().StringVarP(&logDuration, "duration", "d", "", "Duration of the activity (e.g. 45m, 1h30m)")
	_ = logCmd.MarkFlagRequired("duration")
	rootCmd.AddCommand(logCmd)
}

func runLog(_ *cobra.Command, args []string) error {
	cfg, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	dur, err := time.ParseDuration(logDuration)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w (use e.g. 45m, 1h)", err)
	}

	description := strings.Join(args, " ")

	// Use regex redaction
	if cfg.Privacy.Redact {
		patterns := cfg.Privacy.Patterns
		if len(patterns) == 0 {
			patterns = []string{
				`sk-[a-zA-Z0-9]{48}`,
				`ghp_[a-zA-Z0-9]{36}`,
				`xoxb-[0-9]{11,13}-[a-zA-Z0-9]{24}`,
				`AKIA[0-9A-Z]{16}`,
				`sk-ant-api03-[a-zA-Z0-9\-_]{93}`,
				`AIza[0-9A-Za-z\-_]{35}`,
				`Bearer\s+[a-zA-Z0-9\-\._~\+\/]+=*`,
				`eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9\.[a-zA-Z0-9\-_]+\.[a-zA-Z0-9\-_]+`,
			}
		}
		for _, p := range patterns {
			if re, err := regexp.Compile(p); err == nil {
				description = re.ReplaceAllString(description, "[REDACTED]")
			} else {
				fmt.Printf("warning: invalid privacy redaction pattern %q: %v\n", p, err)
			}
		}
	}

	endTime := time.Now().UTC()
	if logTime != "" {
		t, err := time.ParseInLocation("2006-01-02 15:04:05", logTime, time.Local)
		if err != nil {
			return fmt.Errorf("invalid time format: %w", err)
		}
		endTime = t.UTC()
	}

	startTime := endTime.Add(-dur)

	gapMinutes := cfg.Sessions.GapThresholdMinutes
	if gapMinutes <= 0 {
		gapMinutes = 30
	}
	// step by half the gap to ensure retro joins them
	step := time.Duration(gapMinutes) * time.Minute / 2
	if step <= 0 {
		step = 15 * time.Minute
	}

	var timestamps []time.Time
	for cur := startTime; cur.Before(endTime); cur = cur.Add(step) {
		timestamps = append(timestamps, cur)
	}
	timestamps = append(timestamps, endTime)

	rc := classifier.NewRuleClassifier()
	for _, r := range cfg.Classifier.CustomRules {
		rc.AddRule(r.Category, r.Keywords, r.Patterns, r.Priority)
	}

	var lastCategory string

	for _, t := range timestamps {
		e := &models.PromptEvent{
			Agent:     "manual",
			Prompt:    description,
			Timestamp: t,
			Project:   logProject,
		}

		if err := s.InsertPromptEvent(context.Background(), e); err != nil {
			return fmt.Errorf("insert event: %w", err)
		}

		if logCategory != "" {
			c := &models.Classification{
				PromptEventID: e.ID,
				Category:      logCategory,
				Confidence:    1.0,
				Classifier:    "manual",
				CreatedAt:     time.Now().UTC(),
			}
			if err := s.InsertClassification(context.Background(), c); err != nil {
				fmt.Printf("warning: failed to insert classification: %v\n", err)
			}
			lastCategory = logCategory
		} else {
			cs, err := rc.Classify(context.Background(), []models.PromptEvent{*e})
			if err == nil && len(cs) > 0 {
				if err := s.InsertClassification(context.Background(), &cs[0]); err != nil {
					fmt.Printf("warning: failed to auto-classify: %v\n", err)
				} else {
					lastCategory = cs[0].Category
				}
			}
		}
	}

	fmt.Printf("Logged activity: %s\n", description)
	fmt.Printf("  Duration: %s (%s - %s)\n",
		dur,
		startTime.Local().Format("Jan 02 15:04"),
		endTime.Local().Format("Jan 02 15:04"),
	)

	if lastCategory != "" {
		fmt.Printf("  Category: %s\n", lastCategory)
	}

	return nil
}
