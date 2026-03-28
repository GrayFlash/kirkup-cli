package cmd

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayFlash/kirkup-cli/classifier"
	"github.com/GrayFlash/kirkup-cli/models"
)

var (
	logProject  string
	logTime     string
	logCategory string
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
	rootCmd.AddCommand(logCmd)
}

func runLog(_ *cobra.Command, args []string) error {
	cfg, s, cleanup, err := openApp()
	if err != nil {
		return err
	}
	defer cleanup()

	description := args[0]

	// Use regex redaction
	if cfg.Privacy.Redact {
		patterns := cfg.Privacy.Patterns
		if len(patterns) == 0 {
			patterns = []string{`sk-[a-zA-Z0-9]{48}`, `ghp_[a-zA-Z0-9]{36}`, `xoxb-[0-9]{11,13}-[a-zA-Z0-9]{24}`}
		}
		for _, p := range patterns {
			if re, err := regexp.Compile(p); err == nil {
				description = re.ReplaceAllString(description, "[REDACTED]")
			}
		}
	}

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

	// Automatic classification if category provided or via rules
	if logCategory != "" {
		c := &models.Classification{
			PromptEventID: e.ID,
			Category:      logCategory,
			Confidence:    1.0,
			Classifier:    "manual",
			CreatedAt:     time.Now().UTC(),
		}
		_ = s.InsertClassification(context.Background(), c)
	} else {
		// Run rule-based classification immediately for this single event
		rc := classifier.NewRuleClassifier()
		for _, r := range cfg.Classifier.CustomRules {
			rc.AddRule(r.Category, r.Keywords, r.Patterns, r.Priority)
		}
		cs, err := rc.Classify(context.Background(), []models.PromptEvent{*e})
		if err == nil && len(cs) > 0 {
			_ = s.InsertClassification(context.Background(), &cs[0])
			fmt.Printf("Auto-categorised as: %s\n", cs[0].Category)
		}
	}

	fmt.Printf("Logged activity: %s\n", description)
	return nil
}
